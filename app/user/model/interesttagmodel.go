package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ InterestTagModel = (*customInterestTagModel)(nil)

type (
	// InterestTagModel is an interface to be customized, add more methods here,
	// and implement the added methods in customInterestTagModel.
	InterestTagModel interface {
		interestTagModel
	}

	customInterestTagModel struct {
		*defaultInterestTagModel
	}
)

// NewInterestTagModel returns a model for the database table.
func NewInterestTagModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) InterestTagModel {
	return &customInterestTagModel{
		defaultInterestTagModel: newInterestTagModel(conn, c, opts...),
	}
}
