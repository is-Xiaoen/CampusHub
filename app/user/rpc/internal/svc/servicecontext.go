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
	"activity-platform/app/user/rpc/internal/ocr"

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

	// ==================== OCR 服务 ====================

	// OcrFactory OCR提供商工厂
	OcrFactory *ocr.ProviderFactory
}

// NewServiceContext 创建服务上下文
// 初始化所有依赖并注入
func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	db := initDB(c)

	// 初始化Redis连接
	rdb := initRedis(c)

	// 初始化OCR工厂
	ocrFactory := initOcrFactory(c, rdb)

	return &ServiceContext{
		Config: c,
		DB:     db,
		Redis:  rdb,

		// 注入 Model
		UserCreditModel:          model.NewUserCreditModel(db),
		CreditLogModel:           model.NewCreditLogModel(db),
		StudentVerificationModel: model.NewStudentVerificationModel(db),

		// 注入 OCR 工厂
		OcrFactory: ocrFactory,
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
		Addr:         c.Redis.Host,
		Password:     c.Redis.Pass,
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

// initOcrFactory 初始化OCR工厂
func initOcrFactory(c config.Config, rdb *redis.Client) *ocr.ProviderFactory {
	var primary, fallback ocr.Provider

	// 初始化腾讯云OCR（主提供商）
	if c.Ocr.Tencent.Enabled {
		tencentProvider, err := ocr.NewTencentProvider(ocr.TencentConfig{
			Enabled:   c.Ocr.Tencent.Enabled,
			SecretId:  c.Ocr.Tencent.SecretId,
			SecretKey: c.Ocr.Tencent.SecretKey,
			Region:    c.Ocr.Tencent.Region,
			Endpoint:  c.Ocr.Tencent.Endpoint,
			Timeout:   c.Ocr.Tencent.Timeout,
		})
		if err != nil {
			logx.Errorf("初始化腾讯云OCR失败: %v", err)
		} else {
			primary = tencentProvider
			logx.Info("腾讯云OCR初始化成功")
		}
	}

	// 初始化阿里云OCR（备用提供商）
	if c.Ocr.Aliyun.Enabled {
		aliyunProvider, err := ocr.NewAliyunProvider(ocr.AliyunConfig{
			Enabled:         c.Ocr.Aliyun.Enabled,
			AccessKeyId:     c.Ocr.Aliyun.AccessKeyId,
			AccessKeySecret: c.Ocr.Aliyun.AccessKeySecret,
			Endpoint:        c.Ocr.Aliyun.Endpoint,
			Timeout:         c.Ocr.Aliyun.Timeout,
		})
		if err != nil {
			logx.Errorf("初始化阿里云OCR失败: %v", err)
		} else {
			fallback = aliyunProvider
			logx.Info("阿里云OCR初始化成功")
		}
	}

	// 如果主提供商为空，使用备用作为主
	if primary == nil && fallback != nil {
		primary = fallback
		fallback = nil
	}

	if primary == nil {
		logx.Infof("[WARN] OCR服务未配置任何提供商")
		return nil
	}

	return ocr.NewProviderFactory(primary, fallback, rdb)
}
