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

	"activity-platform/app/user/model"
	"activity-platform/app/user/mq/internal/config"

	"github.com/go-redis/redis/v8"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ServiceContext MQ服务上下文
// 包含所有依赖：数据库、缓存、Model等
type ServiceContext struct {
	// Config 服务配置
	Config config.Config

	// DB GORM数据库连接
	DB *gorm.DB

	// Redis Redis客户端
	Redis *redis.Client

	// ==================== Model 层 ====================

	// UserCreditModel 用户信用分数据访问层
	UserCreditModel model.IUserCreditModel

	// CreditLogModel 信用变更记录数据访问层
	CreditLogModel model.ICreditLogModel

	// StudentVerificationModel 学生认证数据访问层
	StudentVerificationModel model.IStudentVerificationModel
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	db := initDB(c)
	rdb := initRedis(c)

	return &ServiceContext{
		Config: c,
		DB:     db,
		Redis:  rdb,

		// 注入 Model
		UserCreditModel:          model.NewUserCreditModel(db),
		CreditLogModel:           model.NewCreditLogModel(db),
		StudentVerificationModel: model.NewStudentVerificationModel(db),
	}
}

// initDB 初始化GORM数据库连接
func initDB(c config.Config) *gorm.DB {
	gormLogger := logger.Default.LogMode(logger.Warn)

	db, err := gorm.Open(mysql.Open(c.MySQL.DataSource), &gorm.Config{
		Logger:                 gormLogger,
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		logx.Errorf("MQ服务连接数据库失败: %v", err)
		panic(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		logx.Errorf("获取数据库实例失败: %v", err)
		panic(err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logx.Info("MQ服务数据库连接初始化成功")
	return db
}

// initRedis 初始化Redis连接
func initRedis(c config.Config) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Host,
		Password:     c.Redis.Pass,
		DB:           0,
		PoolSize:     50,
		MinIdleConns: 5,
	})

	logx.Info("MQ服务Redis连接初始化成功")
	return rdb
}
