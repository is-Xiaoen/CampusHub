package svc

import (
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/internal/config"
	"activity-platform/app/user/rpc/client/creditservice"
	"activity-platform/app/user/rpc/client/verifyservice"

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
	TagModel                  *model.TagModel
	StatusLogModel            *model.ActivityStatusLogModel
	ActivityRegistrationModel *model.ActivityRegistrationModel
	ActivityTicketModel       *model.ActivityTicketModel

	// RPC 客户端（调用其他微服务）
	CreditRpc     creditservice.CreditService // 信用分服务
	VerifyService verifyservice.VerifyService // 学生认证服务
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 1. 初始化数据库连接
	db := initDB(c.MySQL.DataSource)

	// 2. 初始化 Redis
	rds := initRedis(c.Redis.RedisConf)

	// 3. 初始化限流/熔断
	registrationLimiter := limit.NewTokenLimiter(
		c.RegistrationLimit.Rate,
		c.RegistrationLimit.Burst,
		rds,
		"activity:registration:limiter",
	)
	registrationBreaker := breaker.NewBreaker(
		breaker.WithName(c.RegistrationBreaker.Name),
	)

	// 3. 初始化 User RPC 客户端
	userRpcClient := zrpc.MustNewClient(c.UserRpc)
	creditRpc := creditservice.NewCreditService(userRpcClient)
	verifyRpc := verifyservice.NewVerifyService(userRpcClient)

	// 4. 返回 ServiceContext
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
		CategoryModel:             model.NewCategoryModel(db),
		TagModel:                  model.NewTagModel(db),
		StatusLogModel:            model.NewActivityStatusLogModel(db),
		ActivityRegistrationModel: model.NewActivityRegistrationModel(db),
		ActivityTicketModel:       model.NewActivityTicketModel(db),

		// RPC 客户端
		CreditRpc:     creditRpc,
		VerifyService: verifyRpc,
	}
}

// 初始化函数

// initDB 初始化数据库连接
func initDB(dataSource string) *gorm.DB {
	db, err := gorm.Open(mysql.Open(dataSource), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
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
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logx.Info("数据库连接成功")
	return db
}

// initRedis 初始化 Redis 连接
func initRedis(c redis.RedisConf) *redis.Redis {
	rds := redis.MustNewRedis(c)
	logx.Info("Redis 连接成功")
	return rds
}
