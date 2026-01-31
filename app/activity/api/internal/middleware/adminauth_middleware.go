// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import "net/http"

type AdminAuthMiddleware struct {
}

func NewAdminAuthMiddleware() *AdminAuthMiddleware {
	return &AdminAuthMiddleware{}
}

func (m *AdminAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO generate middleware implement function, delete after code implementation

		// Passthrough to next handler if need
		next(w, r)
	}
}
