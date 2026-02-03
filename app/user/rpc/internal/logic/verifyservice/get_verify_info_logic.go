/**
 * @projectName: CampusHub
 * @package: verifyservicelogic
 * @className: GetVerifyInfoLogic
 * @author: lijunqi
 * @description: 获取已通过的认证信息逻辑层
 * @date: 2026-01-31
 * @version: 1.0
 */

package verifyservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/constants"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// GetVerifyInfoLogic 获取已通过的认证信息逻辑处理器
type GetVerifyInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewGetVerifyInfoLogic 创建获取认证信息逻辑实例
func NewGetVerifyInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetVerifyInfoLogic {
	return &GetVerifyInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetVerifyInfo 获取已通过的认证信息
// 业务逻辑:
//   - 仅返回已通过认证用户的脱敏信息
//   - 用于个人中心-认证信息页面展示
func (l *GetVerifyInfoLogic) GetVerifyInfo(in *pb.GetVerifyInfoReq) (*pb.GetVerifyInfoResp, error) {
	// 1. 参数校验
	if in.UserId <= 0 {
		l.Errorf("GetVerifyInfo 参数错误: userId=%d", in.UserId)
		return nil, errorx.ErrInvalidParams("用户ID无效")
	}

	// 2. 查询认证记录
	verification, err := l.svcCtx.StudentVerificationModel.FindByUserID(l.ctx, in.UserId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			l.Infof("GetVerifyInfo 无认证记录: userId=%d", in.UserId)
			return &pb.GetVerifyInfoResp{
				IsVerified: false,
			}, nil
		}
		l.Errorf("GetVerifyInfo 查询失败: userId=%d, err=%v", in.UserId, err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 检查是否已通过认证
	if verification.Status != constants.VerifyStatusPassed {
		l.Infof("GetVerifyInfo 用户未通过认证: userId=%d, status=%d", in.UserId, verification.Status)
		return &pb.GetVerifyInfoResp{
			IsVerified: false,
		}, nil
	}

	// 4. 构建响应（脱敏处理）
	resp := BuildMaskedVerifyInfo(verification)

	l.Infof("GetVerifyInfo 查询成功: userId=%d, school=%s",
		in.UserId, verification.SchoolName)

	return resp, nil
}
