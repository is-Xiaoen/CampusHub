// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	Auth struct {
		AccessSecret string
		AccessExpire int64
	}
}
