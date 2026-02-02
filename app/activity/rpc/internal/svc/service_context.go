package svc

import (
	"fmt"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/internal/config"
	"activity-platform/app/user/rpc/client/creditservice"
	"activity-platform/app/user/rpc/client/tagservice"
	"activity-platform/app/user/rpc/client/verifyservice"

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

	// Model 层
	ActivityModel    *model.ActivityModel
	CategoryModel    *model.CategoryModel
	ActivityTagModel *model.ActivityTagModel       // 活动-标签关联表操作
	TagCacheModel    *model.TagCacheModel          // 标签缓存（从用户服务同步）
	TagStatsModel    *model.ActivityTagStatsModel  // 活动标签统计
	StatusLogModel   *model.ActivityStatusLogModel

	// RPC 客户端（调用其他微服务）
	CreditRpc creditservice.CreditService // 信用分服务
	VerifyRpc verifyservice.VerifyService // 学生认证服务
	TagRpc    tagservice.TagService       // 标签服务（用于同步标签数据）
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 1. 初始化数据库连接
	db := initDB(c.MySQL)

	// 2. 初始化 Redis
	rds := initRedis(c.Redis)

	// 3. 初始化 User RPC 客户端（所有 User 服务共用同一个连接）
	userRpcClient := zrpc.MustNewClient(c.UserRpc)
	creditRpc := creditservice.NewCreditService(userRpcClient)
	verifyRpc := verifyservice.NewVerifyService(userRpcClient)
	tagRpc := tagservice.NewTagService(userRpcClient) // 标签服务客户端

	// 4. 返回 ServiceContext
	return &ServiceContext{
		Config: c,

		// 数据存储
		DB:    db,
		Redis: rds,

		// Model 层
		ActivityModel:    model.NewActivityModel(db),
		CategoryModel:    model.NewCategoryModel(db),
		ActivityTagModel: model.NewActivityTagModel(db),      // 活动-标签关联
		TagCacheModel:    model.NewTagCacheModel(db),         // 标签缓存
		TagStatsModel:    model.NewActivityTagStatsModel(db), // 标签统计
		StatusLogModel:   model.NewActivityStatusLogModel(db),

		// RPC 客户端
		CreditRpc: creditRpc,
		VerifyRpc: verifyRpc,
		TagRpc:    tagRpc,
	}
}

// 初始化函数

// initDB 初始化数据库连接
func initDB(c config.MySQLConfig) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)

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
	sqlDB.SetMaxOpenConns(c.MaxOpenConns)
	sqlDB.SetMaxIdleConns(c.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(c.ConnMaxLifetime) * time.Second)

	logx.Info("数据库连接成功")
	return db
}

// initRedis 初始化 Redis 连接
func initRedis(c redis.RedisConf) *redis.Redis {
	rds := redis.MustNewRedis(c)
	logx.Info("Redis 连接成功")
	return rds
}
