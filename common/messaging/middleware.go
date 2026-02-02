package messaging

// Middleware 定义了中间件的签名
// 中间件可以在消息处理前后执行额外逻辑（如日志、追踪、重试）
// 参数:
//   - next: 下一个处理器
// 返回:
//   - HandlerFunc: 包装后的处理器
type Middleware func(next HandlerFunc) HandlerFunc

// Chain 将多个中间件链接成一个
func Chain(middlewares ...Middleware) Middleware {
	return func(final HandlerFunc) HandlerFunc {
		// 从后向前应用中间件
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}
