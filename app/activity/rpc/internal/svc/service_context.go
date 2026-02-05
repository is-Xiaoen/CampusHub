package svc

import (
	"fmt"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/internal/cache"
	"activity-platform/app/activity/rpc/internal/config"
	"activity-platform/app/activity/rpc/internal/search"
	"activity-platform/app/user/rpc/client/creditservice"
	"activity-platform/app/user/rpc/client/tagservice"
	"activity-platform/app/user/rpc/client/verifyservice"
	"activity-platform/common/breakerx"

	"github.com/zeromicro/go-zero/core/breaker"
	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ServiceContext struct {
	Config config.Config

	// 数据存储
	DB    *gorm.DB     // MySQL 连接
	Redis *redis.Redis // Redis 客户端

	// 高并发、熔断限流组件
	RegistrationLimiter *limit.TokenLimiter
	RegistrationBreaker breaker.Breaker

	// Model 层
	ActivityModel             *model.ActivityModel
	CategoryModel             *model.CategoryModel
	ActivityTagModel          *model.ActivityTagModel      // 活动-标签关联表操作
	TagCacheModel             *model.TagCacheModel         // 标签缓存（从用户服务同步）
	TagModel                  *model.TagModel              // 活动标签查询（兼容层）
	TagStatsModel             *model.ActivityTagStatsModel // 活动标签统计
	StatusLogModel            *model.ActivityStatusLogModel
	ActivityRegistrationModel *model.ActivityRegistrationModel
	ActivityTicketModel       *model.ActivityTicketModel

	// ==================== 缓存服务 ====================
	ActivityCache *cache.ActivityCache // 活动详情缓存
	CategoryCache *cache.CategoryCache // 分类列表缓存
	HotCache      *cache.HotCache      // 热门活动缓存

	// ==================== ES 搜索服务 ====================
	ESClient    *search.ESClientWithBreaker // ES 客户端（带熔断器）
	SyncService *search.SyncService         // ES 数据同步服务

	// RPC 客户端（调用其他微服务）
	CreditRpc     creditservice.CreditService // 信用分服务
	VerifyService verifyservice.VerifyService // 学生认证服务
	VerifyRpc     verifyservice.VerifyService // 学生认证服务
	TagRpc        tagservice.TagService       // 标签服务（用于同步标签数据）
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 1. 初始化数据库连接
	db := initDB(c.MySQL)

	// 2. 初始化业务 Redis（缓存、分布式锁等）
	rds := initRedis(c.BizRedis)

	// 3. 初始化限流/熔断
	registrationLimiter := limit.NewTokenLimiter(
		c.RegistrationLimit.Rate,
		c.RegistrationLimit.Burst,
		rds,
		"activity:registration:limiter",
	)
	registrationBreaker := breakerx.NewSREBreaker(breakerx.SREConfig{
		Name:      c.RegistrationBreaker.Name,
		Requests:  c.RegistrationBreaker.Requests,
		ErrorRate: c.RegistrationBreaker.ErrorRate,
		Timeout:   time.Duration(c.RegistrationBreaker.Timeout) * time.Second,
	})

	// 4. 初始化 User RPC 客户端（所有 User 服务共用同一个连接）
	userRpcClient := zrpc.MustNewClient(c.UserRpc)
	creditRpc := creditservice.NewCreditService(userRpcClient)
	verifyRpc := verifyservice.NewVerifyService(userRpcClient)
	tagRpc := tagservice.NewTagService(userRpcClient) // 标签服务客户端

	// 5. 初始化缓存服务
	activityCache := cache.NewActivityCache(rds, db)
	categoryCache := cache.NewCategoryCache(rds, db)
	hotCache := cache.NewHotCache(rds, db)

	// 6. 初始化 Model 层（提前初始化，供 ES 同步服务使用）
	tagCacheModel := model.NewTagCacheModel(db)
	categoryModel := model.NewCategoryModel(db)

	// 7. 初始化 ES 搜索服务（可选）
	var esClient *search.ESClientWithBreaker
	var syncService *search.SyncService

	if c.Elasticsearch.Enabled {
		var err error
		esClient, err = search.NewESClientWithBreaker(search.ESConfig{
			Enabled:       c.Elasticsearch.Enabled,
			Hosts:         c.Elasticsearch.Hosts,
			Username:      c.Elasticsearch.Username,
			Password:      c.Elasticsearch.Password,
			IndexName:     c.Elasticsearch.IndexName,
			MaxRetries:    c.Elasticsearch.MaxRetries,
			HealthTimeout: c.Elasticsearch.HealthTimeout,
		})
		if err != nil {
			logx.Errorf("[ServiceContext] ES 初始化失败: %v，搜索将降级到 MySQL", err)
			// 不 panic，降级处理：ESClient 为 nil，搜索接口降级到 MySQL
		} else if esClient != nil && esClient.IsEnabled() {
			// ES 初始化成功，创建同步服务
			syncService = search.NewSyncService(esClient.ESClient, db, tagCacheModel, categoryModel)
			logx.Info("[ServiceContext] ES 搜索服务初始化成功")
		}
	} else {
		logx.Info("[ServiceContext] ES 未启用，搜索将使用 MySQL LIKE")
	}

	// 8. 返回 ServiceContext
	return &ServiceContext{
		Config: c,

		// 数据存储
		DB:    db,
		Redis: rds,

		// 限流/熔断
		RegistrationLimiter: registrationLimiter,
		RegistrationBreaker: registrationBreaker,

		// Model 层
		ActivityModel:             model.NewActivityModel(db),
		CategoryModel:             categoryModel,
		ActivityTagModel:          model.NewActivityTagModel(db), // 活动-标签关联
		TagCacheModel:             tagCacheModel,                 // 标签缓存
		TagStatsModel:             model.NewActivityTagStatsModel(db), // 标签统计
		StatusLogModel:            model.NewActivityStatusLogModel(db),
		TagModel:                  model.NewTagModel(db),
		ActivityRegistrationModel: model.NewActivityRegistrationModel(db),
		ActivityTicketModel:       model.NewActivityTicketModel(db),

		// 缓存服务
		ActivityCache: activityCache,
		CategoryCache: categoryCache,
		HotCache:      hotCache,

		// ES 搜索服务
		ESClient:    esClient,
		SyncService: syncService,

		// RPC 客户端
		CreditRpc:     creditRpc,
		VerifyService: verifyRpc,
		TagRpc:        tagRpc,
	}
}

// 初始化函数

// initDB 初始化数据库连接
func initDB(mysqlConf config.MySQLConfig) *gorm.DB {
	dsn := buildMySQLDSN(mysqlConf)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 开发环境打印 SQL
	})
	if err != nil {
		logx.Errorf("连接数据库失败: %v", err)
		panic(err)
	}

	// 设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	maxOpenConns := mysqlConf.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = 100
	}
	maxIdleConns := mysqlConf.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 10
	}
	connMaxLifetime := mysqlConf.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = 3600
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Second)

	logx.Info("数据库连接成功")
	return db
}

// initRedis 初始化 Redis 连接
func initRedis(c redis.RedisConf) *redis.Redis {
	rds := redis.MustNewRedis(c)
	logx.Info("Redis 连接成功")
	return rds
}

func buildMySQLDSN(c config.MySQLConfig) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}
