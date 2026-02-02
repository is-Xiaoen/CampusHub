package interceptor

import (
	"context"

	"CampusHub/common/ctxdata"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ClientTraceIDInterceptor 客户端追踪 ID 拦截器
// 在 RPC 调用时自动将 trace_id 从 context 传递到服务端
//
// 工作流程：
// 1. 从 context 中提取 trace_id
// 2. 通过 gRPC metadata 传递给服务端
//
// 使用方式：
//
//	userRpc := zrpc.MustNewClient(c.UserRpc,
//	    zrpc.WithUnaryClientInterceptor(interceptor.ClientTraceIDInterceptor()))
func ClientTraceIDInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		// 1. 从 context 中提取 trace_id
		traceID := ctxdata.GetTraceIDFromCtx(ctx)

		// 2. 如果有 trace_id，通过 metadata 传递
		if traceID != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "trace_id", traceID)
		}

		// 3. 调用 RPC 方法
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// ServerTraceIDInterceptor 服务端追踪 ID 拦截器
// 在 RPC 服务端自动从 metadata 中提取 trace_id 并注入到 context
//
// 工作流程：
// 1. 从 gRPC metadata 中提取 trace_id
// 2. 注入到 context 中
// 3. 传递给业务处理器
//
// 使用方式：
//
//	server := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
//	    // 注册服务...
//	})
//	server.AddUnaryInterceptors(interceptor.ServerTraceIDInterceptor())
func ServerTraceIDInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {

		// 1. 从 metadata 中提取 trace_id
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if traceIDs := md.Get("trace_id"); len(traceIDs) > 0 {
				// 2. 注入到 context
				ctx = ctxdata.WithTraceID(ctx, traceIDs[0])
			}
		}

		// 3. 调用业务处理器
		return handler(ctx, req)
	}
}

// ClientStreamTraceIDInterceptor 客户端流式追踪 ID 拦截器
// 用于流式 RPC 调用
func ClientStreamTraceIDInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
		method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {

		// 从 context 中提取 trace_id
		traceID := ctxdata.GetTraceIDFromCtx(ctx)

		// 如果有 trace_id，通过 metadata 传递
		if traceID != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "trace_id", traceID)
		}

		return streamer(ctx, desc, cc, method, opts...)
	}
}

// ServerStreamTraceIDInterceptor 服务端流式追踪 ID 拦截器
// 用于流式 RPC 服务
func ServerStreamTraceIDInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
		handler grpc.StreamHandler) error {

		ctx := ss.Context()

		// 从 metadata 中提取 trace_id
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if traceIDs := md.Get("trace_id"); len(traceIDs) > 0 {
				ctx = ctxdata.WithTraceID(ctx, traceIDs[0])
			}
		}

		// 包装 ServerStream
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

// wrappedServerStream 包装 ServerStream 以注入新的 context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context 返回包装后的 context
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
