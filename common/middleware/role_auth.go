package middleware

import (
	"net/http"

	"activity-platform/common/ctxdata"
	"activity-platform/common/errorx"
	"activity-platform/common/response"
	"activity-platform/common/utils/jwt"

	"gorm.io/gorm"
)

type RoleAuthMiddleware struct {
	db   *gorm.DB
	role jwt.Role
}

func NewAdminRoleMiddleware(db *gorm.DB) *RoleAuthMiddleware {
	return &RoleAuthMiddleware{db: db, role: jwt.RoleAdmin}
}

func NewUserRoleMiddleware(db *gorm.DB) *RoleAuthMiddleware {
	return &RoleAuthMiddleware{db: db, role: jwt.RoleUser}
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

		var status int8
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
