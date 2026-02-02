/**
 * @projectName: CampusHub
 * @package: rpcserver
 * @className: error_interceptor
 * @author: lijunqi
 * @description: RPC 服务端错误拦截器，将 BizError 转换为 gRPC Status
 * @date: 2026-01-31
 * @version: 1.0
 */

package rpcserver

import (
	"context"

	"activity-platform/common/errorx"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorInterceptor RPC 服务端错误拦截器
// 作用：将业务错误 BizError 转换为 gRPC Status，使客户端能正确解析错误码
//
// 工作原理：
//  1. 执行业务逻辑（handler）
//  2. 如果返回错误，检查是否是 *BizError
//  3. 如果是 BizError，转换为 gRPC Status，保留业务错误码
//  4. 如果不是 BizError（如数据库错误），返回通用错误，不暴露内部细节
func ErrorInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	// 执行业务逻辑
	resp, err = handler(ctx, req)

	if err != nil {
		// 获取原始错误（支持 errors.Wrap 包装的错误）
		causeErr := errors.Cause(err)

		if bizErr, ok := causeErr.(*errorx.BizError); ok {
			// 自定义业务错误：转换为 gRPC Status
			// 使用业务错误码作为 gRPC code，错误消息作为 message
			logx.WithContext(ctx).Errorf("【RPC-SRV-ERR】 method=%s, code=%d, msg=%s, err=%+v",
				info.FullMethod, bizErr.Code, bizErr.Message, err)

			return nil, status.Error(codes.Code(bizErr.Code), bizErr.Message)
		}

		// 非业务错误（如数据库错误、网络错误等）：记录完整堆栈，返回通用错误
		logx.WithContext(ctx).Errorf("【RPC-SRV-ERR】 method=%s, err=%+v", info.FullMethod, err)

		// 不暴露内部错误详情给客户端，返回通用的内部错误
		return nil, status.Error(codes.Code(errorx.CodeInternalError), "内部服务器错误")
	}

	return resp, nil
}
