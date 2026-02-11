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

	"activity-platform/app/activity/rpc/activityservice"
	"activity-platform/app/user/cache"
	"activity-platform/app/user/model"
	"activity-platform/app/user/ocr"
	"activity-platform/app/user/rpc/internal/config"
	"activity-platform/common/messaging"

	"github.com/go-redis/redis/v8"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
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

	// ==================== Cache 层 ====================

	// CreditCache 信用分缓存服务
	CreditCache cache.ICreditCache

	// VerifyCache 认证状态缓存服务
	VerifyCache cache.IVerifyCache

	// ==================== Model 层 ====================

	// UserModel 用户基础信息数据访问层
	UserModel model.IUserModel

	// UserInterestRelationModel 用户兴趣标签关联数据访问层
	UserInterestRelationModel model.IUserInterestRelationModel

	// InterestTagModel 兴趣标签数据访问层
	InterestTagModel model.IInterestTagModel

	// UserCreditModel 用户信用分数据访问层
	UserCreditModel model.IUserCreditModel

	// CreditLogModel 信用变更记录数据访问层
	CreditLogModel model.ICreditLogModel

	// StudentVerificationModel 学生认证数据访问层
	StudentVerificationModel model.IStudentVerificationModel
	// SysImageModel 图片资源中心数据访问层
	SysImageModel model.ISysImageModel

	// ==================== RPC 服务 ====================

	// ActivityRpc 活动服务 RPC 客户端
	ActivityRpc activityservice.ActivityService

	// ==================== OCR 服务 ====================

	// OcrFactory OCR提供商工厂
	OcrFactory *ocr.ProviderFactory

	// ==================== 消息客户端 ====================

	// MsgClient Watermill 消息客户端（用于发布认证事件到 MQ）
	MsgClient *messaging.Client
}

// NewServiceContext 创建服务上下文
// 初始化所有依赖并注入
// 返回 ServiceContext 和 error，由调用方决定如何处理错误
func NewServiceContext(c config.Config) (*ServiceContext, error) {
	// 初始化数据库连接
	db, err := initDB(c)
	if err != nil {
		return nil, err
	}

	// 初始化Redis连接
	rdb, err := initRedis(c)
	if err != nil {
		return nil, err
	}

	// 初始化OCR工厂（可选，失败不影响服务启动）
	ocrFactory := initOcrFactory(c, rdb)

	// 初始化消息客户端（可选，失败不影响服务启动）
	msgClient := initMsgPublisher(c)

	// 初始化 Activity RPC 客户端（可选，失败不影响服务启动）
	var activityRpc activityservice.ActivityService
	activityRpcClient, err := zrpc.NewClient(c.ActivityRpc)
	if err != nil {
		logx.Errorf("Activity RPC 连接失败（非致命）: %v", err)
	} else {
		activityRpc = activityservice.NewActivityService(activityRpcClient)
		logx.Info("Activity RPC 连接初始化成功")
	}

	return &ServiceContext{
		Config: c,
		DB:     db,
		Redis:  rdb,

		// 注入 Cache
		CreditCache: cache.NewCreditCache(rdb),
		VerifyCache: cache.NewVerifyCache(rdb),

		// 注入 Model
		UserModel:                 model.NewUserModel(db),
		UserInterestRelationModel: model.NewUserInterestRelationModel(db),
		InterestTagModel:          model.NewInterestTagModel(db),
		UserCreditModel:           model.NewUserCreditModel(db),
		CreditLogModel:            model.NewCreditLogModel(db),
		StudentVerificationModel:  model.NewStudentVerificationModel(db),
		SysImageModel:             model.NewSysImageModel(db),

		// 注入 RPC 客户端（可能为 nil）
		ActivityRpc: activityRpc,

		// 注入 OCR 工厂
		OcrFactory: ocrFactory,

		// 注入消息客户端
		MsgClient: msgClient,
	}, nil
}

// initDB 初始化GORM数据库连接
// 配置连接池、日志等
func initDB(c config.Config) (*gorm.DB, error) {
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
		return nil, err
	}

	// 获取底层 sql.DB 以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		logx.Errorf("获取数据库实例失败: %v", err)
		return nil, err
	}

	// 连接池配置
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logx.Info("数据库连接初始化成功")
	return db, nil
}

// initRedis 初始化Redis连接
// 使用 go-zero 的 RedisConf 配置
func initRedis(c config.Config) (*redis.Client, error) {
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
	//     return nil, err
	// }

	logx.Info("Redis连接初始化成功")
	return rdb, nil
}

// initMsgPublisher 初始化消息客户端（可选，失败不阻塞服务启动）
// RPC 服务仅使用其 Publish 能力，不需要 Subscribe/Run
func initMsgPublisher(c config.Config) *messaging.Client {
	// 如果未配置 Redis 地址，跳过初始化
	if c.Messaging.Redis.Addr == "" {
		logx.Infof("[WARN] 消息客户端未配置，跳过初始化")
		return nil
	}

	client, err := messaging.NewClient(messaging.Config{
		Redis: messaging.RedisConfig{
			Addr:     c.Messaging.Redis.Addr,
			Password: c.Messaging.Redis.Password,
			DB:       c.Messaging.Redis.DB,
		},
		ServiceName:  "user-rpc",
		EnableGoZero: true,
	})
	if err != nil {
		logx.Errorf("消息客户端初始化失败: %v", err)
		return nil
	}

	logx.Info("消息客户端初始化成功")
	return client
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
