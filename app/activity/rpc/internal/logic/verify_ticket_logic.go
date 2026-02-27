package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/app/user/rpc/client/tagservice"
	"activity-platform/common/messaging"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/metadata"
)

type VerifyTicketLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewVerifyTicketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifyTicketLogic {
	return &VerifyTicketLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// VerifyTicket 核销票券
func (l *VerifyTicketLogic) VerifyTicket(in *activity.VerifyTicketRequest) (*activity.VerifyTicketResponse, error) {
	userID := in.GetUserId()
	activityID := in.GetActivityId()
	ticketCode := strings.TrimSpace(in.GetTicketCode())
	if userID <= 0 || activityID <= 0 || ticketCode == "" {
		return &activity.VerifyTicketResponse{Result: "fail"}, nil
	}

	// 1) 查询票据
	ticket, err := l.svcCtx.ActivityTicketModel.FindByCode(l.ctx, ticketCode)
	if err != nil {
		if errors.Is(err, model.ErrTicketNotFound) {
			return &activity.VerifyTicketResponse{Result: "fail"}, nil
		}
		l.Errorf("核销查询票据失败: activityId=%d, ticketCode=%s, err=%v", activityID, ticketCode, err)
		return &activity.VerifyTicketResponse{Result: "fail"}, nil
	}
	if ticket.ActivityID != uint64(activityID) {
		return &activity.VerifyTicketResponse{Result: "fail"}, nil
	}
	if ticket.Status != model.TicketStatusUnused {
		return &activity.VerifyTicketResponse{Result: "fail"}, nil
	}

	// 2) 查询活动并校验时间窗口
	activityInfo, err := l.svcCtx.ActivityModel.FindByID(l.ctx, ticket.ActivityID)
	if err != nil {
		if errors.Is(err, model.ErrActivityNotFound) {
			return &activity.VerifyTicketResponse{Result: "fail"}, nil
		}
		l.Errorf("核销查询活动失败: activityId=%d, ticketCode=%s, err=%v", activityID, ticketCode, err)
		return &activity.VerifyTicketResponse{Result: "fail"}, nil
	}
	windowStart, windowEnd, ok := resolveVerifyWindow(ticket, activityInfo)
	if !ok {
		return &activity.VerifyTicketResponse{Result: "fail"}, nil
	}
	now := time.Now()
	nowUnix := now.Unix()
	if nowUnix < windowStart || nowUnix > windowEnd {
		return &activity.VerifyTicketResponse{Result: "fail"}, nil
	}

	// 3) TOTP 校验（如启用）
	if ticket.TotpEnabled {
		if !verifyTicketTotp(ticket, in.GetTotpCode(), now) {
			return &activity.VerifyTicketResponse{Result: "fail"}, nil
		}
	}

	// 4) 标记核销（幂等防重）
	snapshot := buildVerifySnapshot(activityInfo, ticket, nowUnix)
	l.Infof("[VerifyTicket] 开始核销: TicketID=%d, CurrentStatus=%d, TargetStatus=%d, Time=%d",
		ticket.ID, ticket.Status, model.TicketStatusUsed, nowUnix)

	err = l.svcCtx.ActivityTicketModel.MarkUsed(
		l.ctx,
		ticket.ID,
		nowUnix,
		activityInfo.Location,
		snapshot,
	)
	if err == nil {
		l.Infof("[VerifyTicket] 核销成功: TicketID=%d, NewStatus=%d", ticket.ID, model.TicketStatusUsed)
	}
	if err != nil {
		if errors.Is(err, model.ErrTicketNotFound) {
			// 票券不存在或状态已变更（已核销/已作废等）
			l.Infof("核销失败: 票券状态已变更或不存在, activityId=%d, ticketCode=%s, ticketId=%d",
				activityID, ticketCode, ticket.ID)
			return &activity.VerifyTicketResponse{Result: "fail"}, nil
		}
		l.Errorf("核销更新票据失败: activityId=%d, ticketCode=%s, err=%v", activityID, ticketCode, err)
		return &activity.VerifyTicketResponse{Result: "fail"}, nil
	}

	// 异步发布签到信用事件
	l.svcCtx.MsgProducer.PublishCreditEvent(
		l.ctx, messaging.CreditEventCheckin, int64(ticket.ActivityID), int64(ticket.UserID),
	)

	return &activity.VerifyTicketResponse{Result: "success"}, nil
}

const (
	verifyWindowBefore = 24 * time.Hour
	verifyWindowAfter  = 30 * time.Minute
	verifyTotpSkewStep = 1
)

func resolveVerifyWindow(ticket *model.ActivityTicket, activityInfo *model.Activity) (int64, int64, bool) {
	if ticket == nil || activityInfo == nil {
		return 0, 0, false
	}
	if ticket.ValidStartTime > 0 && ticket.ValidEndTime > 0 {
		if ticket.ValidEndTime < ticket.ValidStartTime {
			return 0, 0, false
		}
		return ticket.ValidStartTime, ticket.ValidEndTime, true
	}
	if activityInfo.ActivityStartTime <= 0 || activityInfo.ActivityEndTime <= 0 {
		return 0, 0, false
	}
	if activityInfo.ActivityEndTime < activityInfo.ActivityStartTime {
		return 0, 0, false
	}

	windowStart := activityInfo.ActivityStartTime - int64(verifyWindowBefore/time.Second)
	windowEnd := activityInfo.ActivityEndTime + int64(verifyWindowAfter/time.Second)
	if windowStart <= 0 || windowEnd <= 0 || windowEnd < windowStart {
		return 0, 0, false
	}
	return windowStart, windowEnd, true
}

func verifyTicketTotp(ticket *model.ActivityTicket, input string, now time.Time) bool {
	if ticket == nil {
		return false
	}
	code := strings.TrimSpace(input)
	if code == "" {
		return false
	}
	secret := ticket.TotpSecret
	if secret == "" {
		secret = deriveTotpSecret(int64(ticket.ActivityID), int64(ticket.UserID), ticket.TicketCode)
	}
	if secret == "" {
		return false
	}

	for offset := -verifyTotpSkewStep; offset <= verifyTotpSkewStep; offset++ {
		t := now.Add(time.Duration(offset*totpStepSeconds) * time.Second)
		expected, err := generateTotpCode(secret, t)
		if err == nil && expected == code {
			return true
		}
	}
	return false
}

func buildVerifySnapshot(activityInfo *model.Activity, ticket *model.ActivityTicket, verifyTime int64) string {
	if activityInfo == nil || ticket == nil {
		return ""
	}
	payload := map[string]interface{}{
		"activity_id":   ticket.ActivityID,
		"activity_name": activityInfo.Title,
		"activity_time": activityInfo.ActivityStartTime,
		"ticket_code":   ticket.TicketCode,
		"verify_time":   verifyTime,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(data)
}

const (
	// 缓存相关
	recommendCacheTTLSeconds = 300
	recommendMaxPop          = 10000
	recommendMaxCandidates   = 1000
	recommendUserTagSample   = 50
	recommendUserTagMax      = 20

	// 推荐列表缓存键
	recommendListCacheKeyPrefix = "activity:recommend:list_cache:"

	// 评分权重
	tagMatchWeight      = 0.4 // 标签匹配权重
	hotScoreWeight      = 0.3 // 热度权重
	timeRelevanceWeight = 0.3 // 时间相关性权重

	// 距离衰减相关
	distanceDecayFactor = 100.0 // 距离衰减系数(km)，越大衰减越慢
	distanceBonusFactor = 0.3   // 距离加权系数
)

// RecommendActivitiesInput 活动推荐算法入参
type RecommendActivitiesInput struct {
	UserID        int64
	Limit         int
	UserLatitude  float64
	UserLongitude float64
}

// ActivityScoreDTO 活动评分数据传输对象（用于预计算缓存）
type ActivityScoreDTO struct {
	ActivityID    uint64  `json:"activity_id"`
	TotalScore    float64 `json:"total_score"`    // 综合评分
	TagMatch      float64 `json:"tag_match"`      // 标签匹配分
	HotScore      float64 `json:"hot_score"`      // 热度分
	TimeRelevance float64 `json:"time_relevance"` // 时间相关性分
	ViewCount     uint32  `json:"view_count"`     // 浏览量（新用户排序用）
	ActivityTitle string  `json:"activity_title"` // 活动标题（调试用）
}

type scoredActivity struct {
	id    int64
	score float64
}

// RecommendActivities 计算并返回推荐活动ID列表
// 重构后支持：
// 1. 存量用户使用预计算缓存（recommend_list_cache），由定时任务每10分钟更新
// 2. 新用户按浏览量排序
// 3. 实时地理位置二次加权
func (l *VerifyTicketLogic) RecommendActivities(in *RecommendActivitiesInput) ([]int64, error) {
	if in == nil {
		return []int64{}, errors.New("nil recommend input")
	}
	if in.UserID <= 0 {
		return []int64{}, errors.New("invalid user id")
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}

	// 1. 尝试从预计算缓存获取推荐列表（存量用户）
	cachedList, err := l.getRecommendListCache()
	if err == nil && len(cachedList) > 0 {
		// 存量用户：使用预计算的推荐列表 + 实时地理位置加权
		return l.applyLocationWeightingAndSort(cachedList, in.UserLatitude, in.UserLongitude, limit)
	}

	// 2. 缓存未命中，视为新用户：按浏览量排序
	return l.getNewUserRecommendations(in.UserLatitude, in.UserLongitude, limit)
}

// calculateTagScore 标签相似度(Jaccard)
func calculateTagScore(userTags, activityTags []string) float64 {
	if len(userTags) == 0 && len(activityTags) == 0 {
		return 0
	}
	userSet := make(map[string]struct{}, len(userTags))
	for _, t := range userTags {
		userSet[t] = struct{}{}
	}
	activitySet := make(map[string]struct{}, len(activityTags))
	for _, t := range activityTags {
		activitySet[t] = struct{}{}
	}

	intersection := 0
	for tag := range userSet {
		if _, ok := activitySet[tag]; ok {
			intersection++
		}
	}
	union := len(userSet) + len(activitySet) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// calculatePopScore 热度分(对数归一化)
func calculatePopScore(participants uint32, viewCount uint32) float64 {
	pop := float64(participants)*2 + float64(viewCount)
	score := math.Log(1+pop) / math.Log(1+recommendMaxPop)
	return clampScore(score)
}

// calculateLocScore 地理位置分(Haversine + 指数衰减)
func calculateLocScore(userLat, userLng, actLat, actLng float64, hasUserLoc bool) float64 {
	if !hasUserLoc || (actLat == 0 && actLng == 0) {
		return 0.5
	}
	distance := haversineDistance(userLat, userLng, actLat, actLng)
	return math.Exp(-distance / 50.0)
}

// calculateTimeScore 时间分(24小时内=1.0, 超过则指数衰减)
func calculateTimeScore(startAt int64, now time.Time) float64 {
	start := time.Unix(startAt, 0)
	diff := start.Sub(now)
	if diff <= 24*time.Hour {
		return 1.0
	}
	days := diff.Hours() / 24.0
	score := math.Exp(-days / 7.0)
	if score < 0.3 {
		return 0.3
	}
	if score > 0.8 {
		return 0.8
	}
	return score
}

func clampScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371.0
	toRad := func(deg float64) float64 {
		return deg * math.Pi / 180.0
	}
	lat1Rad := toRad(lat1)
	lat2Rad := toRad(lat2)
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

// getUserProfile 获取用户画像（占位：待用户服务完成）
func (l *VerifyTicketLogic) getUserProfile(userID int64) ([]string, float64, float64, bool) {
	if userID <= 0 {
		return []string{}, 0, 0, false
	}
	userTags := l.getUserTagsFromRPC(userID)
	if len(userTags) == 0 {
		userTags = l.getUserTagsFromRegistrations(userID)
	}
	return userTags, 0, 0, false
}

// getActivityTags 获取活动标签（占位：待活动标签关联完善）
func (l *VerifyTicketLogic) getActivityTags(activityID int64) []string {
	if activityID <= 0 {
		return []string{}
	}
	tags, err := l.svcCtx.TagCacheModel.FindByActivityID(l.ctx, uint64(activityID))
	if err != nil {
		l.Infof("[WARNING] 获取活动标签失败: activityId=%d, err=%v", activityID, err)
		return []string{}
	}
	if len(tags) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		result = append(result, tag.Name)
	}
	return normalizeTags(result)
}

// getCandidateActivities 获取候选活动（报名中的活动）
func (l *VerifyTicketLogic) getCandidateActivities(userID int64, limit int) ([]model.Activity, error) {
	if limit <= 0 {
		limit = recommendMaxCandidates
	}

	pageSize := limit
	if pageSize > model.MaxPageSize {
		pageSize = model.MaxPageSize
	}
	if pageSize <= 0 {
		pageSize = model.DefaultPageSize
	}

	listLogic := NewListActivitiesLogic(l.ctx, l.svcCtx)
	ids := make([]uint64, 0, pageSize)
	page := 1
	for len(ids) < limit {
		resp, err := listLogic.ListActivities(&activity.ListActivitiesReq{
			Page:     int32(page),
			PageSize: int32(pageSize),
			Status:   int32(model.StatusPublished), // 2=报名中
			IsAdmin:  false,
			ViewerId: userID,
		})
		if err != nil {
			return nil, err
		}
		if len(resp.List) == 0 {
			break
		}
		for _, item := range resp.List {
			ids = append(ids, uint64(item.Id))
			if len(ids) >= limit {
				break
			}
		}
		if len(resp.List) < pageSize {
			break
		}
		page++
		if page > model.MaxPage {
			break
		}
	}

	if len(ids) == 0 {
		return []model.Activity{}, nil
	}

	activities, err := l.svcCtx.ActivityModel.FindByIDs(l.ctx, ids)
	if err != nil {
		return nil, err
	}
	return activities, nil
}

// getUserTagsFromRPC 通过用户服务获取用户标签
func (l *VerifyTicketLogic) getUserTagsFromRPC(userID int64) []string {
	if l.svcCtx.TagRpc == nil || userID <= 0 {
		return []string{}
	}

	ctx, cancel := context.WithTimeout(l.ctx, 2*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "user_id", fmt.Sprintf("%d", userID))

	resp, err := l.svcCtx.TagRpc.GetUserTags(ctx, &tagservice.GetUserTagsReq{
		UserId: userID,
	})
	if err != nil {
		l.Infof("[WARNING] 获取用户标签失败: userId=%d, err=%v", userID, err)
		return []string{}
	}
	if resp == nil {
		return []string{}
	}

	var names []string
	for _, tag := range resp.GetTags() {
		if tag.GetName() != "" {
			names = append(names, tag.GetName())
		}
	}
	return normalizeTags(names)
}

// getUserTagsFromRegistrations  从报名历史推断用户标签
func (l *VerifyTicketLogic) getUserTagsFromRegistrations(userID int64) []string {
	regs, err := l.svcCtx.ActivityRegistrationModel.ListByUserID(l.ctx, uint64(userID), 0, recommendUserTagSample)
	if err != nil {
		l.Infof("[WARNING] 查询用户报名记录失败: userId=%d, err=%v", userID, err)
		return []string{}
	}
	if len(regs) == 0 {
		return []string{}
	}

	activityIDs := make([]uint64, 0, len(regs))
	seen := make(map[uint64]struct{}, len(regs))
	for _, reg := range regs {
		if _, ok := seen[reg.ActivityID]; ok {
			continue
		}
		seen[reg.ActivityID] = struct{}{}
		activityIDs = append(activityIDs, reg.ActivityID)
	}
	if len(activityIDs) == 0 {
		return []string{}
	}

	tagsMap, err := l.svcCtx.TagCacheModel.FindByActivityIDs(l.ctx, activityIDs)
	if err != nil {
		l.Infof("[WARNING] 查询活动标签失败: userId=%d, err=%v", userID, err)
		return []string{}
	}

	frequency := make(map[string]int)
	for _, tags := range tagsMap {
		for _, tag := range tags {
			name := normalizeTagName(tag.Name)
			if name == "" {
				continue
			}
			frequency[name]++
		}
	}
	if len(frequency) == 0 {
		return []string{}
	}

	type pair struct {
		name  string
		count int
	}
	list := make([]pair, 0, len(frequency))
	for name, count := range frequency {
		list = append(list, pair{name: name, count: count})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].count == list[j].count {
			return list[i].name < list[j].name
		}
		return list[i].count > list[j].count
	})

	limit := recommendUserTagMax
	if limit > len(list) {
		limit = len(list)
	}
	names := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		names = append(names, list[i].name)
	}
	return names
}

// getActivityTagsMap 批量获取活动标签
func (l *VerifyTicketLogic) getActivityTagsMap(activityIDs []uint64) (map[uint64][]string, error) {
	if len(activityIDs) == 0 {
		return map[uint64][]string{}, nil
	}
	tagsMap, err := l.svcCtx.TagCacheModel.FindByActivityIDs(l.ctx, activityIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[uint64][]string, len(tagsMap))
	for activityID, tags := range tagsMap {
		names := make([]string, 0, len(tags))
		for _, tag := range tags {
			names = append(names, tag.Name)
		}
		result[activityID] = normalizeTags(names)
	}
	return result, nil
}

func splitTagNames(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', '，', ';', '；', '|', '/', '\\':
			return true
		default:
			return false
		}
	})
	return parts
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	uniq := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		name := normalizeTagName(tag)
		if name == "" {
			continue
		}
		if _, ok := uniq[name]; ok {
			continue
		}
		uniq[name] = struct{}{}
		result = append(result, name)
	}
	return result
}

func normalizeTagName(tag string) string {
	name := strings.TrimSpace(tag)
	if name == "" {
		return ""
	}
	return strings.ToLower(name)
}

// ==================== 重构后的推荐相关方法 ====================

// getRecommendListCache 获取预计算的推荐列表缓存
// 返回的缓存由定时任务每10分钟更新一次
func (l *VerifyTicketLogic) getRecommendListCache() ([]ActivityScoreDTO, error) {
	cacheKey := recommendListCacheKeyPrefix + "global"
	cached, err := l.svcCtx.Redis.Get(cacheKey)
	if err != nil {
		return nil, err
	}

	cached = strings.TrimSpace(cached)
	if cached == "" {
		return nil, errors.New("recommend list cache not found")
	}

	var list []ActivityScoreDTO
	if jsonErr := json.Unmarshal([]byte(cached), &list); jsonErr != nil {
		l.Errorf("[RecommendActivities] 解析推荐列表缓存失败: err=%v", jsonErr)
		return nil, jsonErr
	}

	if len(list) == 0 {
		return nil, errors.New("empty recommend list cache")
	}

	return list, nil
}

// applyLocationWeightingAndSort 应用地理位置加权并返回最终推荐结果
// 对于存量用户，在预计算分数基础上加上距离加权分
func (l *VerifyTicketLogic) applyLocationWeightingAndSort(cachedList []ActivityScoreDTO, userLat, userLng float64, limit int) ([]int64, error) {
	// 如果没有提供用户位置，直接按预计算分数返回
	if userLat == 0 || userLng == 0 {
		result := make([]int64, 0, limit)
		for i := 0; i < limit && i < len(cachedList); i++ {
			result = append(result, int64(cachedList[i].ActivityID))
		}
		return result, nil
	}

	// 计算每个活动的最终分数 = 预计算分数 + 距离加权分
	type finalScoredActivity struct {
		id         int64
		finalScore float64
	}
	finalScores := make([]finalScoredActivity, len(cachedList))

	for i, item := range cachedList {
		// 获取活动详情以获取经纬度
		activity, err := l.svcCtx.ActivityModel.FindByID(l.ctx, item.ActivityID)
		if err != nil {
			// 如果查询失败，使用原分数
			finalScores[i] = finalScoredActivity{
				id:         int64(item.ActivityID),
				finalScore: item.TotalScore,
			}
			continue
		}

		// 计算距离加权分
		distanceBonus := l.calculateDistanceBonus(userLat, userLng, activity.Latitude, activity.Longitude)

		// 最终分数 = 预计算分数 + 距离加权分
		finalScore := item.TotalScore + distanceBonus

		finalScores[i] = finalScoredActivity{
			id:         int64(item.ActivityID),
			finalScore: finalScore,
		}
	}

	// 按最终分数降序排序
	sort.Slice(finalScores, func(i, j int) bool {
		return finalScores[i].finalScore > finalScores[j].finalScore
	})

	// 返回前 N 个
	result := make([]int64, 0, limit)
	for i := 0; i < limit && i < len(finalScores); i++ {
		result = append(result, finalScores[i].id)
	}

	return result, nil
}

// calculateDistanceBonus 计算距离加权分
// 使用 Redis GEODIST 获取距离，距离越近分数越高
func (l *VerifyTicketLogic) calculateDistanceBonus(userLat, userLng, actLat, actLng float64) float64 {
	// 如果活动没有位置信息，返回默认分数
	if actLat == 0 || actLng == 0 {
		return 0
	}

	// 使用本地计算距离（Haversine公式）
	distance := haversineDistance(userLat, userLng, actLat, actLng)

	// 距离加权分：距离越近分数越高，使用指数衰减
	// bonus = distanceBonusFactor * exp(-distance / distanceDecayFactor)
	bonus := distanceBonusFactor * math.Exp(-distance/distanceDecayFactor)

	return bonus
}

// getNewUserRecommendations 新用户推荐逻辑
// 按浏览量（view_count）从高到低排序，直接返回
func (l *VerifyTicketLogic) getNewUserRecommendations(userLat, userLng float64, limit int) ([]int64, error) {
	// 查询已发布的活动，按浏览量降序排序
	activities, err := l.svcCtx.ActivityModel.FindPublishedOrderByViewCount(l.ctx, limit)
	if err != nil {
		l.Errorf("[RecommendActivities] 查询新用户推荐活动失败: err=%v", err)
		return nil, err
	}

	if len(activities) == 0 {
		return []int64{}, nil
	}

	// 直接返回按浏览量排序的活动ID列表
	result := make([]int64, 0, len(activities))
	for _, act := range activities {
		result = append(result, int64(act.ID))
	}

	return result, nil
}
