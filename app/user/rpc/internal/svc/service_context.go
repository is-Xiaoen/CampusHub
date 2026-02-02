/**
 * @projectName: CampusHub
 * @package: svc
 * @className: ServiceContext
 * @author: lijunqi
 * @description: 用户RPC服务上下文，负责依赖注入
 * @date: 2026-01-30
 * @version: 1.0
 */

package svc

import (
	"time"

	"activity-platform/app/user/model"
	"activity-platform/app/user/rpc/internal/config"

	"github.com/go-redis/redis/v8"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ServiceContext 用户服务上下文
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
// 初始化所有依赖并注入
func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	db := initDB(c)

	// 初始化Redis连接
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
// 配置连接池、日志等
func initDB(c config.Config) *gorm.DB {
	// GORM 日志配置
	// 使用默认的 logger，设置为 Warn 级别（只记录慢查询和错误）
	gormLogger := logger.Default.LogMode(logger.Warn)

	// 连接数据库
	db, err := gorm.Open(mysql.Open(c.MySQL.DataSource), &gorm.Config{
		Logger:                 gormLogger,
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		logx.Errorf("连接数据库失败: %v", err)
		panic(err)
	}

	// 获取底层 sql.DB 以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		logx.Errorf("获取数据库实例失败: %v", err)
		panic(err)
	}

	// 连接池配置
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logx.Info("数据库连接初始化成功")
	return db
}

// initRedis 初始化Redis连接
// 使用 go-zero 的 RedisConf 配置
func initRedis(c config.Config) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         c.BizRedis.Host,
		Password:     c.BizRedis.Pass,
		DB:           0,
		PoolSize:     100,
		MinIdleConns: 10,
	})

	// 测试连接（可选，生产环境可去掉以加快启动）
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()
	// if err := rdb.Ping(ctx).Err(); err != nil {
	//     logx.Errorf("Redis连接失败: %v", err)
	//     panic(err)
	// }

	logx.Info("Redis连接初始化成功")
	return rdb
}
