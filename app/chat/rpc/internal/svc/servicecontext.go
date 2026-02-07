package svc

import (
	"fmt"
	"log"
	"time"

	"activity-platform/app/chat/model"
	"activity-platform/app/chat/rpc/internal/config"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/messaging"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ServiceContext 聊天服务上下文
type ServiceContext struct {
	Config config.Config

	// 数据库
	DB *gorm.DB

	// Redis
	RedisClient *redis.Client

	// 消息中间件客户端
	MsgClient *messaging.Client

	// ==================== User RPC 客户端（供 MQ 消费者使用）====================

	// UserCreditRpc User 信用分服务 RPC 客户端
	UserCreditRpc pb.CreditServiceClient

	// UserVerifyRpc User 认证服务 RPC 客户端
	UserVerifyRpc pb.VerifyServiceClient

	// Model 层
	GroupModel        model.GroupModel
	GroupMemberModel  model.GroupMemberModel
	MessageModel      model.MessageModel
	NotificationModel model.NotificationModel
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	db, err := initDB(c)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 初始化 Redis 连接
	redisClient := initRedis(c.CacheRedis.Host, c.CacheRedis.Pass, c.CacheRedis.DB)

	// 初始化消息中间件客户端
	msgClient, err := initMessaging(c)
	if err != nil {
		log.Fatalf("消息中间件初始化失败: %v", err)
	}

	// 初始化 User RPC 客户端（信用分 + 认证 共用同一个连接）
	userRpcConn := zrpc.MustNewClient(c.UserRpc).Conn()

	return &ServiceContext{
		Config:            c,
		DB:                db,
		RedisClient:       redisClient,
		MsgClient:         msgClient,
		UserCreditRpc:     pb.NewCreditServiceClient(userRpcConn),
		UserVerifyRpc:     pb.NewVerifyServiceClient(userRpcConn),
		GroupModel:        model.NewGroupModel(db),
		GroupMemberModel:  model.NewGroupMemberModel(db),
		MessageModel:      model.NewMessageModel(db),
		NotificationModel: model.NewNotificationModel(db),
	}
}

// initDB 初始化数据库连接
func initDB(c config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(c.MySQL.DataSource), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	// 获取底层的 sql.DB 对象，配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取 sql.DB 失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期

	return db, nil
}

// initRedis 初始化 Redis 连接
func initRedis(host, pass string, db int) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     host,
		Password: pass,
		DB:       db,
	})

	return client
}

// initMessaging 初始化消息中间件
func initMessaging(c config.Config) (*messaging.Client, error) {
	// 构建 messaging 配置
	msgConfig := messaging.Config{
		Redis: messaging.RedisConfig{
			Addr:     c.CacheRedis.Host,
			Password: c.CacheRedis.Pass,
			DB:       c.CacheRedis.DB,
		},
		ServiceName:   c.Messaging.ServiceName,
		EnableMetrics: c.Messaging.EnableMetrics,
		EnableGoZero:  c.Messaging.EnableGoZero,
		RetryConfig: messaging.RetryConfig{
			MaxRetries:      c.Messaging.Retry.MaxRetries,
			InitialInterval: c.Messaging.Retry.InitialInterval,
			MaxInterval:     c.Messaging.Retry.MaxInterval,
			Multiplier:      c.Messaging.Retry.Multiplier,
		},
		DLQConfig: messaging.DLQConfig{
			Enabled:     true,
			TopicSuffix: ".dlq",
		},
		// 配置订阅者选项以避免 XPENDING 语法错误
		// 如果 Redis 版本 < 6.2.0，可以禁用 ClaimInterval 来避免 XPENDING 调用
		SubscriberConfig: messaging.SubscriberConfig{
			ClaimInterval:      0,                // 设置为 0 禁用自动声明（避免 XPENDING 错误）
			NackResendInterval: time.Second * 10, // 保持 NACK 重发
			MaxIdleTime:        time.Minute * 5,  // 保持空闲时间检查
		},
	}

	// 创建消息客户端
	client, err := messaging.NewClient(msgConfig)
	if err != nil {
		return nil, fmt.Errorf("创建消息客户端失败: %w", err)
	}

	return client, nil
}
