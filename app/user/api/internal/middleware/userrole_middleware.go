// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import "net/http"

type UserRoleMiddleware struct {
}

func NewUserRoleMiddleware() *UserRoleMiddleware {
	return &UserRoleMiddleware{}
}

func (m *UserRoleMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO generate middleware implement function, delete after code implementation

		// Passthrough to next handler if need
		next(w, r)
	}
}
