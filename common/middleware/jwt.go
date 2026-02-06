package middleware

import (
	"net/http"
	"strings"

	"activity-platform/common/ctxdata"
	"activity-platform/common/errorx"
	"activity-platform/common/response"
	"activity-platform/common/utils/jwt"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type RoleAuthMiddleware struct {
	db           *gorm.DB
	redis        *redis.Client
	accessSecret string
	role         jwt.Role
}

func NewAdminRoleMiddleware(db *gorm.DB, redis *redis.Client, accessSecret string) *RoleAuthMiddleware {
	return &RoleAuthMiddleware{db: db, redis: redis, accessSecret: accessSecret, role: jwt.RoleAdmin}
}

func NewUserRoleMiddleware(db *gorm.DB, redis *redis.Client, accessSecret string) *RoleAuthMiddleware {
	return &RoleAuthMiddleware{db: db, redis: redis, accessSecret: accessSecret, role: jwt.RoleUser}
}

func (m *RoleAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.db == nil {
			response.Fail(w, errorx.ErrInternalError())
			return
		}

		ctx := r.Context()
		userId := ctxdata.GetUserIDFromCtx(ctx)
		if userId <= 0 {
			response.Fail(w, errorx.ErrUnauthorized())
			return
		}

		if m.role == jwt.RoleAdmin && !jwt.IsAdmin(ctx) {
			response.Fail(w, errorx.ErrForbidden())
			return
		}

		if m.role == jwt.RoleUser && !jwt.IsUser(ctx) {
			response.Fail(w, errorx.ErrForbidden())
			return
		}

		// 检查黑名单
		token := r.Header.Get("Authorization")
		if token != "" {
			parts := strings.Split(token, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
				isBlacklisted, _ := jwt.CheckTokenBlacklist(r.Context(), m.redis, token, m.accessSecret)
				if isBlacklisted {
					response.Fail(w, errorx.ErrInvalidToken())
					return
				}
			}
		}

		var status int64
		err := m.db.WithContext(ctx).
			Table("users").
			Select("status").
			Where("user_id = ?", userId).
			Take(&status).Error
		if err != nil {
			response.Fail(w, errorx.ErrDBError(err))
			return
		}

		if status != 1 {
			response.Fail(w, errorx.ErrForbidden())
			return
		}

		next(w, r.WithContext(ctx))
	}
}
