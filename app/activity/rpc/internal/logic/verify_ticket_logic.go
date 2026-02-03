package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/app/user/rpc/client/tagservice"

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
	// todo: add your logic here and delete this line

	return &activity.VerifyTicketResponse{}, nil
}

const (
	recommendCacheTTLSeconds = 300
	recommendMaxPop          = 10000
	recommendMaxCandidates   = 1000
	recommendUserTagSample   = 50
	recommendUserTagMax      = 20
)

// RecommendActivitiesInput 活动推荐算法入参
type RecommendActivitiesInput struct {
	UserID        int64
	Limit         int
	UserLatitude  float64
	UserLongitude float64
}

type scoredActivity struct {
	id    int64
	score float64
}

// RecommendActivities 计算并返回推荐活动ID列表
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

	cacheKey := fmt.Sprintf("activity:recommend:%d:%d", in.UserID, limit)
	if in.UserLatitude != 0 || in.UserLongitude != 0 {
		cacheKey = fmt.Sprintf("activity:recommend:%d:%d:%.4f:%.4f", in.UserID, limit, in.UserLatitude, in.UserLongitude)
	}
	if cached, err := l.svcCtx.Redis.Get(cacheKey); err == nil {
		cached = strings.TrimSpace(cached)
		if cached != "" {
			var ids []int64
			if jsonErr := json.Unmarshal([]byte(cached), &ids); jsonErr == nil {
				return ids, nil
			}
		}
	}

	userTags, userLat, userLng, hasUserLoc := l.getUserProfile(in.UserID)
	if in.UserLatitude != 0 || in.UserLongitude != 0 {
		userLat = in.UserLatitude
		userLng = in.UserLongitude
		hasUserLoc = true
	}
	activities, err := l.getCandidateActivities(in.UserID, recommendMaxCandidates)
	if err != nil {
		return nil, err
	}
	if len(activities) == 0 {
		return []int64{}, nil
	}

	activityIDs := make([]uint64, 0, len(activities))
	for _, act := range activities {
		activityIDs = append(activityIDs, act.ID)
	}
	tagsMap, err := l.getActivityTagsMap(activityIDs)
	if err != nil {
		l.Infof("[WARNING] 批量获取活动标签失败: err=%v", err)
		tagsMap = nil
	}

	scores := make([]scoredActivity, len(activities))
	workerCount := 16
	if len(activities) < workerCount {
		workerCount = len(activities)
	}

	jobs := make(chan int)
	var wg sync.WaitGroup
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for idx := range jobs {
				act := activities[idx]
				var activityTags []string
				if tagsMap != nil {
					activityTags = tagsMap[act.ID]
				} else {
					activityTags = l.getActivityTags(int64(act.ID))
				}

				tagScore := calculateTagScore(userTags, activityTags)
				popScore := calculatePopScore(act.CurrentParticipants, act.ViewCount)
				locScore := calculateLocScore(userLat, userLng, act.Latitude, act.Longitude, hasUserLoc)
				timeScore := calculateTimeScore(act.ActivityStartTime, time.Now())

				final := tagScore*0.3 + popScore*0.3 + locScore*0.2 + timeScore*0.2
				final = clampScore(final)
				scores[idx] = scoredActivity{id: int64(act.ID), score: final}
			}
		}()
	}
	for i := range activities {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})
	if limit > len(scores) {
		limit = len(scores)
	}
	ids := make([]int64, 0, limit)
	for i := 0; i < limit; i++ {
		ids = append(ids, scores[i].id)
	}

	if data, err := json.Marshal(ids); err == nil {
		_ = l.svcCtx.Redis.Setex(cacheKey, string(data), recommendCacheTTLSeconds)
	}

	return ids, nil
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

	resp, err := l.svcCtx.TagRpc.GetUserTags(ctx, &tagservice.GetUserTagsRep{})
	if err != nil {
		l.Infof("[WARNING] 获取用户标签失败: userId=%d, err=%v", userID, err)
		return []string{}
	}
	if resp == nil {
		return []string{}
	}

	names := splitTagNames(resp.GetName())
	if len(names) == 0 && resp.GetId() > 0 {
		if tag, err := l.svcCtx.TagCacheModel.FindByID(l.ctx, resp.GetId()); err == nil && tag != nil {
			names = []string{tag.Name}
		}
	}
	return normalizeTags(names)
}

// getUserTagsFromRegistrations 从报名历史推断用户标签
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
