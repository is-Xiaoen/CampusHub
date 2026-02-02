package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserInterestRelationModel = (*customUserInterestRelationModel)(nil)

type (
	// UserInterestRelationModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserInterestRelationModel.
	UserInterestRelationModel interface {
		userInterestRelationModel
	}

	customUserInterestRelationModel struct {
		*defaultUserInterestRelationModel
	}
)

// NewUserInterestRelationModel returns a model for the database table.
func NewUserInterestRelationModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserInterestRelationModel {
	return &customUserInterestRelationModel{
		defaultUserInterestRelationModel: newUserInterestRelationModel(conn, c, opts...),
	}
}
