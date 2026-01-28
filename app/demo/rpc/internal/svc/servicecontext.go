// ============================================================================
// 服务上下文（Service Context）
// ============================================================================

package svc

import (
	"activity-platform/app/demo/rpc/internal/config"
	"activity-platform/app/demo/rpc/internal/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ServiceContext 服务上下文
type ServiceContext struct {
	Config    config.Config
	DB        *gorm.DB
	ItemModel *model.ItemModel
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	db := initDB(c.MySQL.DataSource)
	return &ServiceContext{
		Config:    c,
		DB:        db,
		ItemModel: model.NewItemModel(db),
	}
}

// initDB 初始化数据库
func initDB(dsn string) *gorm.DB {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Info),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		logx.Severef("连接数据库失败: %v", err)
		panic(err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	logx.Info("数据库连接成功")
	return db
}
