/**
 * @projectName: CampusHub
 * @package: svc
 * @className: ServiceContext
 * @author: lijunqi
 * @description: 用户MQ服务上下文，负责依赖注入
 * @date: 2026-01-30
 * @version: 1.0
 */

package svc

import (
	"time"

	"activity-platform/app/user/cache"
	"activity-platform/app/user/model"
	"activity-platform/app/user/mq/internal/config"
	"activity-platform/common/messaging"

	"github.com/go-redis/redis/v8"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ServiceContext MQ服务上下文
// 包含所有依赖：数据库、缓存、Model、消息客户端等
type ServiceContext struct {
	// Config 服务配置
	Config config.Config

	// DB GORM数据库连接
	DB *gorm.DB

	// Redis Redis客户端（用于缓存操作）
	Redis *redis.Client

	// MsgClient Watermill 消息客户端
	MsgClient *messaging.Client

	// ==================== Cache 层 ====================

	// CreditCache 信用分缓存服务
	CreditCache cache.ICreditCache

	// VerifyCache 认证状态缓存服务
	VerifyCache cache.IVerifyCache

	// ==================== Model 层 ====================

	// UserCreditModel 用户信用分数据访问层
	UserCreditModel model.IUserCreditModel

	// CreditLogModel 信用变更记录数据访问层
	CreditLogModel model.ICreditLogModel

	// StudentVerificationModel 学生认证数据访问层
	StudentVerificationModel model.IStudentVerificationModel
}

// NewServiceContext 创建服务上下文
// 返回 ServiceContext 和 error，由调用方决定如何处理错误
func NewServiceContext(c config.Config) (*ServiceContext, error) {
	db, err := initDB(c)
	if err != nil {
		return nil, err
	}

	rdb, err := initRedis(c)
	if err != nil {
		return nil, err
	}

	msgClient, err := initMessaging(c)
	if err != nil {
		return nil, err
	}

	return &ServiceContext{
		Config:    c,
		DB:        db,
		Redis:     rdb,
		MsgClient: msgClient,

		// 注入 Cache
		CreditCache: cache.NewCreditCache(rdb),
		VerifyCache: cache.NewVerifyCache(rdb),

		// 注入 Model
		UserCreditModel:          model.NewUserCreditModel(db),
		CreditLogModel:           model.NewCreditLogModel(db),
		StudentVerificationModel: model.NewStudentVerificationModel(db),
	}, nil
}

// initDB 初始化GORM数据库连接
func initDB(c config.Config) (*gorm.DB, error) {
	gormLogger := logger.Default.LogMode(logger.Warn)

	db, err := gorm.Open(mysql.Open(c.MySQL.DataSource), &gorm.Config{
		Logger:                 gormLogger,
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		logx.Errorf("MQ服务连接数据库失败: %v", err)
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		logx.Errorf("获取数据库实例失败: %v", err)
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logx.Info("MQ服务数据库连接初始化成功")
	return db, nil
}

// initRedis 初始化Redis连接（用于缓存）
func initRedis(c config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         c.BizRedis.Host,
		Password:     c.BizRedis.Pass,
		DB:           0,
		PoolSize:     50,
		MinIdleConns: 5,
	})

	logx.Info("MQ服务Redis连接初始化成功")
	return rdb, nil
}

// initMessaging 初始化 Watermill 消息客户端
func initMessaging(c config.Config) (*messaging.Client, error) {
	msgConfig := messaging.Config{
		// ==================== Redis 连接配置 ====================
		Redis: messaging.RedisConfig{
			Addr:     c.Messaging.Redis.Addr,     // Redis 服务器地址，如 "192.168.10.4:6379"
			Password: c.Messaging.Redis.Password, // Redis 密码
			DB:       c.Messaging.Redis.DB,       // Redis 数据库编号（0-15）
		},

		// ==================== 服务配置 ====================
		ServiceName:   c.Name,                    // 服务名称，用于日志、指标标签、消费者组
		EnableMetrics: c.Messaging.EnableMetrics, // 是否启用 Prometheus 指标监控
		EnableGoZero:  c.Messaging.EnableGoZero,  // 是否启用 go-zero trace_id 链路追踪传播

		// ==================== 重试配置（指数退避算法）====================
		// 重试间隔计算: InitialInterval × (Multiplier ^ 重试次数)，最大不超过 MaxInterval
		// 例如: 100ms → 200ms → 400ms → 800ms...（最大 10s）
		RetryConfig: messaging.RetryConfig{
			MaxRetries:      c.Messaging.Retry.MaxRetries,      // 最大重试次数，超过后放弃
			InitialInterval: c.Messaging.Retry.InitialInterval, // 首次重试等待时间
			MaxInterval:     c.Messaging.Retry.MaxInterval,     // 最大重试等待时间上限
			Multiplier:      c.Messaging.Retry.Multiplier,      // 退避倍数，每次重试间隔翻倍
		},
	}

	client, err := messaging.NewClient(msgConfig)
	if err != nil {
		logx.Errorf("MQ服务消息客户端初始化失败: %v", err)
		return nil, err
	}

	logx.Info("MQ服务消息客户端初始化成功")
	return client, nil
}
