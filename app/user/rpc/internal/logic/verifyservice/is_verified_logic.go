/**
 * @projectName: CampusHub
 * @package: verifyservicelogic
 * @className: IsVerifiedLogic
 * @author: lijunqi
 * @description: 查询用户是否已完成学生认证逻辑层
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
func (l *IsVerifiedLogic) IsVerified(in *pb.IsVerifiedReq) (*pb.IsVerifiedResp, error) {
	// 1. 参数校验
	if in.UserId <= 0 {
		l.Errorf("IsVerified 参数错误: userId=%d", in.UserId)
		return nil, errorx.ErrInvalidParams("用户ID无效")
	}

	// 2. 查询认证记录
	verification, err := l.svcCtx.StudentVerificationModel.FindByUserID(l.ctx, in.UserId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			l.Infof("IsVerified 无认证记录: userId=%d", in.UserId)
			return &pb.IsVerifiedResp{
				IsVerified: false,
			}, nil
		}
		l.Errorf("IsVerified 查询失败: userId=%d, err=%v", in.UserId, err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 构建响应（脱敏处理）
	resp := BuildIsVerifiedResp(verification)

	if resp.IsVerified {
		l.Infof("IsVerified 查询成功: userId=%d, isVerified=true, school=%s",
			in.UserId, verification.SchoolName)
	} else {
		l.Infof("IsVerified 用户未通过认证: userId=%d, status=%d", in.UserId, verification.Status)
	}

	return resp, nil
}
