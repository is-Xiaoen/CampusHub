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

	"github.com/zeromicro/go-zero/core/logx"
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
)

type scoredActivity struct {
	id    int64
	score float64
}

// RecommendActivities 计算并返回推荐活动ID列表
func (l *VerifyTicketLogic) RecommendActivities(userID int64, limit int) ([]int64, error) {
	if userID <= 0 {
		return []int64{}, errors.New("invalid user id")
	}
	if limit <= 0 {
		limit = 10
	}

	cacheKey := fmt.Sprintf("activity:recommend:%d:%d", userID, limit)
	if cached, err := l.svcCtx.Redis.Get(cacheKey); err == nil {
		cached = strings.TrimSpace(cached)
		if cached != "" {
			var ids []int64
			if jsonErr := json.Unmarshal([]byte(cached), &ids); jsonErr == nil {
				return ids, nil
			}
		}
	}

	userTags, userLat, userLng, hasUserLoc := l.getUserProfile(userID)
	activities, err := l.getCandidateActivities(userID, recommendMaxCandidates)
	if err != nil {
		return nil, err
	}
	if len(activities) == 0 {
		return []int64{}, nil
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
				activityTags := l.getActivityTags(int64(act.ID))

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
	return []string{}, 0, 0, false
}

// getActivityTags 获取活动标签（占位：待活动标签关联完善）
func (l *VerifyTicketLogic) getActivityTags(activityID int64) []string {
	return []string{}
}

// getCandidateActivities 获取候选活动（占位：待排除逻辑完善）
func (l *VerifyTicketLogic) getCandidateActivities(userID int64, limit int) ([]model.Activity, error) {
	if limit <= 0 {
		limit = recommendMaxCandidates
	}
	now := time.Now().Unix()
	var activities []model.Activity
	err := l.svcCtx.DB.WithContext(l.ctx).
		Model(&model.Activity{}).
		Where("status = ? AND activity_end_time > ?", model.StatusPublished, now).
		Where("NOT EXISTS (SELECT 1 FROM activity_registrations r WHERE r.activity_id = activities.id AND r.user_id = ? AND r.status = ?)",
			userID, model.RegistrationStatusSuccess).
		Limit(limit).
		Find(&activities).Error
	if err != nil {
		return nil, err
	}
	return activities, nil
}
