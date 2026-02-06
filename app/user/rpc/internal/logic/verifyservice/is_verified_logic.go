/**
 * @projectName: CampusHub
 * @package: verifyservicelogic
 * @className: IsVerifiedLogic
 * @author: lijunqi
 * @description: 查询用户是否已完成学生认证逻辑层（含缓存）
 * @date: 2026-01-31
 * @version: 1.0
 */

package verifyservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// IsVerifiedLogic 查询用户是否已完成学生认证逻辑处理器
type IsVerifiedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewIsVerifiedLogic 创建查询认证状态逻辑实例
func NewIsVerifiedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IsVerifiedLogic {
	return &IsVerifiedLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// IsVerified 查询用户是否已完成学生认证
// 业务逻辑:
//   - 供 Activity 服务调用，用于报名/发布活动前校验
//   - 返回认证状态和脱敏后的认证信息
//   - 使用 Cache-Aside 模式缓存认证状态
func (l *IsVerifiedLogic) IsVerified(in *pb.IsVerifiedReq) (*pb.IsVerifiedResp, error) {
	// 1. 参数校验
	if in.UserId <= 0 {
		l.Errorf("IsVerified 参数错误: userId=%d", in.UserId)
		return nil, errorx.ErrInvalidParams("用户ID无效")
	}

	// 2. 先尝试从缓存获取认证状态
	isVerified, exists, cacheErr := l.svcCtx.VerifyCache.Get(l.ctx, in.UserId)
	if cacheErr != nil {
		// Redis 错误，记录日志但继续查库（降级处理）
		l.Errorf("IsVerified Redis读取失败: userId=%d, err=%v", in.UserId, cacheErr)
	}
	if exists {
		// 缓存命中
		l.Infof("IsVerified 缓存命中: userId=%d, isVerified=%v", in.UserId, isVerified)

		if isVerified {
			// 已认证，需要查库获取详细信息（用于返回脱敏数据）
			verification, dbErr := l.svcCtx.StudentVerificationModel.FindByUserID(l.ctx, in.UserId)
			if dbErr == nil {
				return BuildIsVerifiedResp(verification), nil
			}
			// 查库失败，但缓存显示已认证，返回简单的已认证响应
			return &pb.IsVerifiedResp{IsVerified: true}, nil
		}
		// 未认证
		return &pb.IsVerifiedResp{IsVerified: false}, nil
	}

	// 3. 缓存未命中，查询数据库
	verification, err := l.svcCtx.StudentVerificationModel.FindByUserID(l.ctx, in.UserId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			l.Infof("IsVerified 无认证记录: userId=%d", in.UserId)
			// 回填缓存（未认证状态）
			go func() {
				if setErr := l.svcCtx.VerifyCache.Set(l.ctx, in.UserId, false); setErr != nil {
					l.Errorf("IsVerified 回填缓存失败: userId=%d, err=%v", in.UserId, setErr)
				}
			}()
			return &pb.IsVerifiedResp{IsVerified: false}, nil
		}
		l.Errorf("IsVerified 查询失败: userId=%d, err=%v", in.UserId, err)
		return nil, errorx.ErrDBError(err)
	}

	// 4. 构建响应（脱敏处理）
	resp := BuildIsVerifiedResp(verification)

	// 5. 回填缓存
	go func() {
		if setErr := l.svcCtx.VerifyCache.Set(l.ctx, in.UserId, resp.IsVerified); setErr != nil {
			l.Errorf("IsVerified 回填缓存失败: userId=%d, err=%v", in.UserId, setErr)
		}
	}()

	if resp.IsVerified {
		l.Infof("IsVerified 查询成功: userId=%d, isVerified=true, school=%s",
			in.UserId, verification.SchoolName)
	} else {
		l.Infof("IsVerified 用户未通过认证: userId=%d, status=%d", in.UserId, verification.Status)
	}

	return resp, nil
}
