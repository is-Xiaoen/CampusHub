// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"activity-platform/app/user/api/internal/config"
	"activity-platform/app/user/api/internal/middleware"
	"activity-platform/app/user/model"
	"activity-platform/app/user/rpc/client/captchaservice"
	"activity-platform/app/user/rpc/client/creditservice"
	"activity-platform/app/user/rpc/client/qqemail"
	"activity-platform/app/user/rpc/client/tagservice"
	"activity-platform/app/user/rpc/client/userbasicservice"
	"activity-platform/app/user/rpc/client/verifyservice"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ServiceContext struct {
	Config             config.Config
	UserRoleMiddleware rest.Middleware

	Redis *redis.Client
	// DB GORM数据库连接
	DB *gorm.DB

	// UserModel 用户基础信息数据访问层
	UserModel model.IUserModel

	// CaptchaServiceRpc 验证码服务 RPC 客户端
	CaptchaServiceRpc captchaservice.CaptchaService
	// CreditServiceRpc 信用分服务 RPC 客户端
	CreditServiceRpc creditservice.CreditService

	// VerifyServiceRpc 认证服务 RPC 客户端
	VerifyServiceRpc verifyservice.VerifyService

	// TagServiceRpc 标签服务 RPC 客户端
	TagServiceRpc tagservice.TagService

	// UserBasicServiceRpc 用户基础服务 RPC 客户端（登录、注册、忘记密码等）
	UserBasicServiceRpc userbasicservice.UserBasicService

	// QQEmailRpc QQ邮箱服务 RPC 客户端
	QQEmailRpc qqemail.QQEmail
}

func NewServiceContext(c config.Config) *ServiceContext {

	// 创建 User RPC 客户端连接
	userRpcClient := zrpc.MustNewClient(c.UserRpc)

	// 初始化 Redis 客户端
	rdb := initRedis(c)

	// 初始化数据库连接
	db := initDB(c)

	return &ServiceContext{
		Config:             c,
		UserRoleMiddleware: middleware.NewUserRoleMiddleware().Handle,
		Redis:              rdb,
		DB:                 db,
		UserModel:          model.NewUserModel(db),

		// 初始化 RPC 客户端
		CaptchaServiceRpc:   captchaservice.NewCaptchaService(userRpcClient),
		CreditServiceRpc:    creditservice.NewCreditService(userRpcClient),
		VerifyServiceRpc:    verifyservice.NewVerifyService(userRpcClient),
		TagServiceRpc:       tagservice.NewTagService(userRpcClient),
		UserBasicServiceRpc: userbasicservice.NewUserBasicService(userRpcClient),
		QQEmailRpc:          qqemail.NewQQEmail(userRpcClient),
	}
}

// initRedis 初始化Redis客户端
func initRedis(c config.Config) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     c.BizRedis.Host,
		Password: c.BizRedis.Pass,
		DB:       0,
	})
	logx.Info("Redis连接初始化成功")
	return rdb
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
