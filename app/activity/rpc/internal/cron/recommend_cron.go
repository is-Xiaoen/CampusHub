package cron

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"activity-platform/app/activity/model"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
)

// ==================== 常量定义 ====================

const (
	// 分布式锁配置
	recommendLockKey    = "activity:cron:recommend:cache"
	recommendLockExpire = 600 // 锁过期时间（秒），10分钟

	// 默认执行间隔（秒）
	recommendDefaultInterval = 600 // 10分钟

	// 缓存键
	recommendListCacheKeyPrefix = "activity:recommend:list_cache:"

	// 评分权重
	tagMatchWeight      = 0.4 // 标签匹配权重
	hotScoreWeight      = 0.3 // 热度权重
	timeRelevanceWeight = 0.3 // 时间相关性权重
)

// ==================== ActivityScoreDTO 活动评分数据传输对象 ====================

type ActivityScoreDTO struct {
	ActivityID    uint64  `json:"activity_id"`
	TotalScore    float64 `json:"total_score"`
	TagMatch      float64 `json:"tag_match"`
	HotScore      float64 `json:"hot_score"`
	TimeRelevance float64 `json:"time_relevance"`
	ViewCount     uint32  `json:"view_count"`
	ActivityTitle string  `json:"activity_title"`
}

// ==================== RecommendCron 推荐列表缓存定时任务 ====================

// RecommendCron 推荐列表缓存定时任务
//
// 功能说明：
//   - 预计算活动推荐列表并缓存到 Redis
//   - 为存量用户提供基于综合评分的推荐
//
// 执行策略：
//   - 默认每10分钟执行一次
//   - 使用 Redis 分布式锁，确保多实例部署时只有一个实例执行
type RecommendCron struct {
	redis         *redis.Redis
	db            *gorm.DB
	activityModel *model.ActivityModel
	tagStatsModel *model.ActivityTagStatsModel
	tagCacheModel *model.TagCacheModel

	intervalSeconds int           // 执行间隔（秒）
	stopChan        chan struct{} // 停止信号
	running         atomic.Bool   // 运行状态（原子操作，并发安全）
	stopOnce        sync.Once     // 保证 close(stopChan) 只执行一次
	ownerID         string        // 分布式锁 owner 标识（防止误删他人锁）
}

// NewRecommendCron 创建推荐列表缓存定时任务
func NewRecommendCron(
	rds *redis.Redis,
	db *gorm.DB,
	activityModel *model.ActivityModel,
	tagStatsModel *model.ActivityTagStatsModel,
	tagCacheModel *model.TagCacheModel,
) *RecommendCron {
	return &RecommendCron{
		redis:           rds,
		db:              db,
		activityModel:   activityModel,
		tagStatsModel:   tagStatsModel,
		tagCacheModel:   tagCacheModel,
		intervalSeconds: recommendDefaultInterval,
		stopChan:        make(chan struct{}),
		ownerID:         uuid.New().String(),
	}
}

// SetInterval 设置执行间隔（秒）
func (c *RecommendCron) SetInterval(seconds int) {
	if seconds > 0 {
		c.intervalSeconds = seconds
	}
}

// Start 启动定时任务
func (c *RecommendCron) Start() {
	// CAS 操作：只有从 false → true 时才启动，天然防重入
	if !c.running.CompareAndSwap(false, true) {
		logx.Info("[RecommendCron] 定时任务已在运行中，跳过重复启动")
		return
	}

	logx.Infof("[RecommendCron] 启动推荐列表缓存定时任务，执行间隔: %d 秒, owner: %s", c.intervalSeconds, c.ownerID)

	go func() {
		// 启动后立即执行一次
		c.execute()

		ticker := time.NewTicker(time.Duration(c.intervalSeconds) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.execute()
			case <-c.stopChan:
				logx.Info("[RecommendCron] 定时任务已停止")
				return
			}
		}
	}()
}

// Stop 停止定时任务
func (c *RecommendCron) Stop() {
	if !c.running.Load() {
		return
	}
	c.stopOnce.Do(func() {
		close(c.stopChan)
	})
	c.running.Store(false)
}

// execute 执行推荐列表缓存计算
func (c *RecommendCron) execute() {
	ctx := context.Background()

	// 尝试获取分布式锁
	locked, err := c.tryLock(ctx, recommendLockKey)
	if err != nil {
		logx.Errorf("[RecommendCron] 获取锁失败: err=%v", err)
		return
	}
	if !locked {
		// 其他实例正在执行，跳过
		return
	}
	defer c.unlock(ctx, recommendLockKey)

	logx.Info("[RecommendCron] 开始计算推荐列表缓存")

	startTime := time.Now()

	err = c.calculateAndCacheRecommendList(ctx)

	duration := time.Since(startTime)

	if err != nil {
		logx.Errorf("[RecommendCron] 计算推荐列表失败: err=%v, 耗时: %v", err, duration)
		return
	}

	logx.Infof("[RecommendCron] 计算推荐列表成功，耗时: %v", duration)
}

// calculateAndCacheRecommendList 预计算推荐列表并缓存到Redis
func (c *RecommendCron) calculateAndCacheRecommendList(ctx context.Context) error {
	// 1. 获取所有已发布的活动
	activities, err := c.activityModel.FindAllPublished(ctx)
	if err != nil {
		logx.Errorf("[RecommendCron] 查询活动失败: err=%v", err)
		return err
	}

	if len(activities) == 0 {
		logx.Infof("[RecommendCron] 无活动需要计算")
		return nil
	}

	// 2. 获取所有活动ID
	activityIDs := make([]uint64, 0, len(activities))
	for _, act := range activities {
		activityIDs = append(activityIDs, act.ID)
	}

	// 3. 批量获取活动标签
	tagsMap, err := c.getActivityTagsMap(ctx, activityIDs)
	if err != nil {
		logx.Infof("[RecommendCron] 批量获取活动标签失败: err=%v", err)
		tagsMap = make(map[uint64][]string)
	}

	// 4. 获取全局热门标签（作为默认用户标签）
	globalTags := c.getGlobalHotTags(ctx)

	// 5. 计算每个活动的综合评分
	scoredList := make([]ActivityScoreDTO, 0, len(activities))
	maxViewCount := uint32(1)
	maxParticipants := uint32(1)

	// 先计算最大值用于归一化
	for _, act := range activities {
		if act.ViewCount > maxViewCount {
			maxViewCount = act.ViewCount
		}
		if act.CurrentParticipants > maxParticipants {
			maxParticipants = act.CurrentParticipants
		}
	}

	now := time.Now()

	for _, act := range activities {
		activityTags := tagsMap[act.ID]

		// 计算各维度分数
		tagMatch := calculateTagScore(globalTags, activityTags)
		hotScore := calculateHotScore(act.ViewCount, act.CurrentParticipants, maxViewCount, maxParticipants)
		timeRelevance := calculateTimeRelevance(act.ActivityStartTime, now)

		// 综合评分
		totalScore := tagMatch*tagMatchWeight + hotScore*hotScoreWeight + timeRelevance*timeRelevanceWeight

		scoredList = append(scoredList, ActivityScoreDTO{
			ActivityID:    act.ID,
			TotalScore:    totalScore,
			TagMatch:      tagMatch,
			HotScore:      hotScore,
			TimeRelevance: timeRelevance,
			ViewCount:     act.ViewCount,
			ActivityTitle: act.Title,
		})
	}

	// 6. 按综合评分降序排序
	sort.Slice(scoredList, func(i, j int) bool {
		return scoredList[i].TotalScore > scoredList[j].TotalScore
	})

	// 7. 缓存到Redis
	cacheKey := recommendListCacheKeyPrefix + "global"
	data, err := json.Marshal(scoredList)
	if err != nil {
		logx.Errorf("[RecommendCron] 序列化失败: err=%v", err)
		return err
	}

	// 缓存10分钟（600秒）
	err = c.redis.SetexCtx(ctx, cacheKey, string(data), 600)
	if err != nil {
		logx.Errorf("[RecommendCron] 缓存写入失败: err=%v", err)
		return err
	}

	logx.Infof("[RecommendCron] 成功缓存 %d 个活动推荐列表", len(scoredList))
	return nil
}

// getActivityTagsMap 批量获取活动标签
func (c *RecommendCron) getActivityTagsMap(ctx context.Context, activityIDs []uint64) (map[uint64][]string, error) {
	type ActivityTag struct {
		ActivityID uint64 `gorm:"column:activity_id"`
		TagID      uint64 `gorm:"column:tag_id"`
	}

	var tagRelations []ActivityTag
	err := c.db.WithContext(ctx).
		Table("activity_tags").
		Select("activity_id, tag_id").
		Where("activity_id IN ?", activityIDs).
		Find(&tagRelations).Error
	if err != nil {
		return nil, err
	}

	// 获取所有涉及的标签ID
	tagIDs := make([]uint64, 0, len(tagRelations))
	for _, rel := range tagRelations {
		tagIDs = append(tagIDs, rel.TagID)
	}

	// 批量获取标签信息
	tagMap := make(map[uint64]string)
	if len(tagIDs) > 0 {
		var tags []model.TagCache
		err = c.db.WithContext(ctx).
			Where("id IN ?", tagIDs).
			Find(&tags).Error
		if err != nil {
			return nil, err
		}
		for _, tag := range tags {
			tagMap[tag.ID] = tag.Name
		}
	}

	// 构建活动ID -> 标签名列表的映射
	result := make(map[uint64][]string)
	for _, rel := range tagRelations {
		if tagName, ok := tagMap[rel.TagID]; ok {
			result[rel.ActivityID] = append(result[rel.ActivityID], tagName)
		}
	}

	return result, nil
}

// getGlobalHotTags 获取全局热门标签（作为默认用户标签）
func (c *RecommendCron) getGlobalHotTags(ctx context.Context) []string {
	stats, err := c.tagStatsModel.GetHotTags(ctx, 20)
	if err != nil {
		logx.Infof("[RecommendCron] 获取热门标签失败: err=%v", err)
		return []string{}
	}

	tags := make([]string, 0, len(stats))
	for _, stat := range stats {
		tags = append(tags, stat.TagName)
	}
	return tags
}

// ==================== 分布式锁 ====================

// unlockScript Lua 脚本：只有 owner 匹配时才删除锁
const unlockScripts = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end
`

// tryLock 尝试获取分布式锁（带 owner 标识）
func (c *RecommendCron) tryLock(ctx context.Context, key string) (bool, error) {
	// SETNX + EXPIRE 原子操作，value 存入 ownerID
	ok, err := c.redis.SetnxExCtx(ctx, key, c.ownerID, recommendLockExpire)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// unlock 释放分布式锁（仅 owner 匹配时才删除）
func (c *RecommendCron) unlock(ctx context.Context, key string) {
	result, err := c.redis.EvalCtx(ctx, unlockScripts, []string{key}, c.ownerID)
	if err != nil {
		logx.Errorf("[RecommendCron] 释放锁失败: err=%v", err)
		return
	}
	if result.(int64) == 0 {
		logx.Infof("[RecommendCron] 锁已被其他实例持有，跳过释放")
	}
}

// RunOnce 手动执行一次推荐列表缓存计算
func (c *RecommendCron) RunOnce() {
	logx.Info("[RecommendCron] 手动触发推荐列表缓存计算")
	c.execute()
}

// ==================== 评分计算函数 ====================

// calculateTagScore 标签相似度（Jaccard）
func calculateTagScore(userTags, activityTags []string) float64 {
	if len(userTags) == 0 && len(activityTags) == 0 {
		return 0
	}
	userSet := make(map[string]struct{}, len(userTags))
	for _, t := range userTags {
		userSet[normalizeTagName(t)] = struct{}{}
	}
	activitySet := make(map[string]struct{}, len(activityTags))
	for _, t := range activityTags {
		activitySet[normalizeTagName(t)] = struct{}{}
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

// normalizeTagName 标签名称归一化
func normalizeTagName(tag string) string {
	name := strings.TrimSpace(tag)
	if name == "" {
		return ""
	}
	return strings.ToLower(name)
}

// calculateHotScore 计算热度分数（30%权重）
func calculateHotScore(viewCount, participants, maxViewCount, maxParticipants uint32) float64 {
	if maxViewCount == 0 {
		maxViewCount = 1
	}
	if maxParticipants == 0 {
		maxParticipants = 1
	}

	viewScore := float64(viewCount) / float64(maxViewCount)
	participantScore := float64(participants) / float64(maxParticipants)

	hotScore := (viewScore + participantScore) / 2.0

	return clampScore(hotScore)
}

// calculateTimeRelevance 计算时间相关性分数（30%权重）
func calculateTimeRelevance(activityStartTime int64, now time.Time) float64 {
	activityTime := time.Unix(activityStartTime, 0)

	if activityTime.Before(now) {
		return 0.1
	}

	duration := activityTime.Sub(now)

	switch {
	case duration <= 24*time.Hour:
		return 1.0
	case duration <= 7*24*time.Hour:
		days := duration.Hours() / 24.0
		return 1.0 - (days-1.0)/6.0*0.5
	default:
		return 0.3
	}
}

// clampScore 分数限制在 [0, 1] 范围内
func clampScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}
