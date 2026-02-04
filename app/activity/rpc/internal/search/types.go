// Package search ES 搜索模块
//
// 本模块提供 Elasticsearch 搜索功能，包括：
// - ES 客户端初始化（带熔断器）
// - 活动搜索（关键词搜索、过滤、排序）
// - 数据同步（双写、全量同步）
// - 搜索建议（自动补全）
//
// 技术选型：
// - ES 版本：7.17.x（LTS 稳定版）
// - Go 客户端：olivere/elastic v7
// - 中文分词：IK 分词器
package search

import "fmt"

// ==================== ES 文档结构 ====================
//
// ⚠️ 字段类型与 model.Activity 严格对齐
// 时间字段使用 int64 Unix 时间戳（秒级），ES 端配置 format: epoch_second

// ActivityDoc ES 活动文档结构
// 对应 activity-platform/app/activity/model.Activity
type ActivityDoc struct {
	// ===== 基础字段（与 Activity model 完全对齐）=====
	ID          uint64 `json:"id"`          // Activity.ID
	Title       string `json:"title"`       // Activity.Title
	Description string `json:"description"` // Activity.Description
	CategoryID  uint64 `json:"category_id"` // Activity.CategoryID
	Status      int8   `json:"status"`      // Activity.Status

	// ===== 组织者信息 =====
	OrganizerID     uint64 `json:"organizer_id"`     // Activity.OrganizerID
	OrganizerName   string `json:"organizer_name"`   // Activity.OrganizerName
	OrganizerAvatar string `json:"organizer_avatar"` // Activity.OrganizerAvatar

	// ===== 地点信息 =====
	Location      string    `json:"location"`                 // Activity.Location
	AddressDetail string    `json:"address_detail,omitempty"` // Activity.AddressDetail
	GeoLocation   *GeoPoint `json:"geo_location,omitempty"`   // 经纬度（可选）

	// ===== 时间信息（int64 Unix 时间戳）=====
	RegisterStartTime int64 `json:"register_start_time"` // Activity.RegisterStartTime
	RegisterEndTime   int64 `json:"register_end_time"`   // Activity.RegisterEndTime
	ActivityStartTime int64 `json:"activity_start_time"` // Activity.ActivityStartTime
	ActivityEndTime   int64 `json:"activity_end_time"`   // Activity.ActivityEndTime

	// ===== 名额与统计 =====
	MaxParticipants     uint32 `json:"max_participants"`     // Activity.MaxParticipants
	CurrentParticipants uint32 `json:"current_participants"` // Activity.CurrentParticipants
	ViewCount           uint32 `json:"view_count"`           // Activity.ViewCount

	// ===== 冗余字段（需要关联查询填充）=====
	CategoryName string   `json:"category_name,omitempty"` // 从 Category 表查询
	Tags         []string `json:"tags,omitempty"`          // 标签名称数组
	CoverURL     string   `json:"cover_url"`               // Activity.CoverURL
	CoverType    int8     `json:"cover_type"`              // Activity.CoverType

	// ===== 时间戳 =====
	CreatedAt int64 `json:"created_at"` // Activity.CreatedAt
	UpdatedAt int64 `json:"updated_at"` // Activity.UpdatedAt

	// ===== 搜索建议（可选）=====
	Suggest *Suggest `json:"suggest,omitempty"`
}

// GeoPoint 地理坐标
type GeoPoint struct {
	Lat float64 `json:"lat"` // 纬度
	Lon float64 `json:"lon"` // 经度
}

// Suggest 搜索建议结构
type Suggest struct {
	Input    []string          `json:"input"`
	Contexts map[string]string `json:"contexts,omitempty"`
}

// ==================== 搜索请求/响应 ====================

// SearchRequest 搜索请求
type SearchRequest struct {
	Query      string  // 搜索关键词
	CategoryID uint64  // 分类筛选（对应 Activity.CategoryID）
	Status     []int8  // 状态筛选（对应 Activity.Status）
	StartTime  *int64  // 开始时间筛选（Unix 时间戳）
	EndTime    *int64  // 结束时间筛选（Unix 时间戳）
	SortBy     string  // 排序：relevance, time, hot, newest
	Page       int     // 页码
	PageSize   int     // 每页数量
}

// SearchResponse 搜索响应
type SearchResponse struct {
	Total        int64                  `json:"total"`
	Activities   []ActivityDoc          `json:"activities"`
	Aggregations map[string]interface{} `json:"aggregations,omitempty"`
	TookMs       int64                  `json:"took_ms"`
}

// ==================== 搜索建议请求/响应 ====================

// SuggestRequest 搜索建议请求
type SuggestRequest struct {
	Prefix     string // 输入前缀
	CategoryID uint64 // 分类上下文
	Size       int    // 返回数量
}

// SuggestResponse 搜索建议响应
type SuggestResponse struct {
	Suggestions []string `json:"suggestions"`
}

// ==================== 状态常量 ====================

const (
	// 活动状态（与 model.Status* 保持一致）
	StatusDraft     int8 = 0 // 草稿
	StatusPending   int8 = 1 // 待审核
	StatusPublished int8 = 2 // 已发布
	StatusOngoing   int8 = 3 // 进行中
	StatusFinished  int8 = 4 // 已结束
	StatusRejected  int8 = 5 // 已拒绝
	StatusCancelled int8 = 6 // 已取消
)

// PublicStatuses 公开可搜索的状态
var PublicStatuses = []int8{StatusPublished, StatusOngoing, StatusFinished}

// ==================== 索引配置 ====================

const (
	// DefaultIndexName 默认索引名称
	DefaultIndexName = "activities"
)

// IndexMapping 活动索引 Mapping（JSON 格式）
// 使用 IK 分词器，支持中文搜索
const IndexMapping = `{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0,
    "analysis": {
      "analyzer": {
        "ik_smart_synonym": {
          "type": "custom",
          "tokenizer": "ik_smart",
          "filter": ["lowercase", "activity_synonym"]
        }
      },
      "filter": {
        "activity_synonym": {
          "type": "synonym",
          "synonyms": [
            "跑步,夜跑,晨跑,马拉松",
            "讲座,分享会,沙龙,论坛",
            "比赛,竞赛,大赛,锦标赛",
            "志愿,义工,公益,志愿者"
          ]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id": {"type": "unsigned_long"},
      "title": {
        "type": "text",
        "analyzer": "ik_max_word",
        "search_analyzer": "ik_smart",
        "fields": {
          "keyword": {"type": "keyword", "ignore_above": 256}
        }
      },
      "description": {
        "type": "text",
        "analyzer": "ik_max_word",
        "search_analyzer": "ik_smart"
      },
      "category_id": {"type": "unsigned_long"},
      "category_name": {"type": "keyword"},
      "status": {"type": "byte"},
      "organizer_id": {"type": "unsigned_long"},
      "organizer_name": {"type": "keyword"},
      "organizer_avatar": {"type": "keyword", "index": false},
      "location": {
        "type": "text",
        "analyzer": "ik_smart",
        "fields": {"keyword": {"type": "keyword"}}
      },
      "address_detail": {"type": "text", "analyzer": "ik_smart"},
      "geo_location": {"type": "geo_point"},
      "register_start_time": {"type": "date", "format": "epoch_second"},
      "register_end_time": {"type": "date", "format": "epoch_second"},
      "activity_start_time": {"type": "date", "format": "epoch_second"},
      "activity_end_time": {"type": "date", "format": "epoch_second"},
      "max_participants": {"type": "integer"},
      "current_participants": {"type": "integer"},
      "view_count": {"type": "integer"},
      "tags": {"type": "keyword"},
      "cover_url": {"type": "keyword", "index": false},
      "cover_type": {"type": "byte"},
      "created_at": {"type": "date", "format": "epoch_second"},
      "updated_at": {"type": "date", "format": "epoch_second"},
      "suggest": {
        "type": "completion",
        "analyzer": "ik_smart",
        "contexts": [{"name": "category", "type": "category"}]
      }
    }
  }
}`

// ==================== 工具函数 ====================

// DocID 生成文档 ID
func DocID(activityID uint64) string {
	return fmt.Sprintf("%d", activityID)
}
