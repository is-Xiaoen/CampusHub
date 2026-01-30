/**
 * @projectName: CampusHub
 * @package: creditservicelogic
 * @className: GetCreditInfoLogic
 * @author: lijunqi
 * @description: 获取信用信息逻辑层
 * @date: 2026-01-30
 * @version: 1.0
 */

package creditservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/constants"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// GetCreditInfoLogic 获取信用信息逻辑处理器
type GetCreditInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewGetCreditInfoLogic 创建获取信用信息逻辑实例
func NewGetCreditInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCreditInfoLogic {
	return &GetCreditInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetCreditInfo 获取用户信用信息
// 业务逻辑:
//   - 查询用户当前的信用分、等级、特权信息
//   - 如果用户没有信用记录，返回错误
func (l *GetCreditInfoLogic) GetCreditInfo(in *pb.GetCreditInfoReq) (*pb.GetCreditInfoResp, error) {
	// 1. 参数校验
	if in.UserId <= 0 {
		l.Errorf("GetCreditInfo 参数错误: userId=%d", in.UserId)
		return nil, errorx.ErrInvalidParams("用户ID无效")
	}

	// 2. 查询信用记录
	credit, err := l.svcCtx.UserCreditModel.FindByUserID(l.ctx, in.UserId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			l.Infof("GetCreditInfo 信用记录不存在: userId=%d", in.UserId)
			return nil, errorx.ErrCreditNotFound()
		}
		l.Errorf("GetCreditInfo 查询信用记录失败: userId=%d, err=%v", in.UserId, err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 计算权限
	canPublish := credit.Score >= constants.CreditThresholdPublish
	canParticipate := credit.Score >= constants.CreditThresholdBlacklist

	// 4. 获取等级名称
	levelName := constants.GetCreditLevelName(credit.Level)

	l.Infof("GetCreditInfo 查询成功: userId=%d, score=%d, level=%d",
		in.UserId, credit.Score, credit.Level)

	return &pb.GetCreditInfoResp{
		UserId:         credit.UserID,
		Score:          int64(credit.Score),
		Level:          int32(credit.Level),
		LevelName:      levelName,
		CanPublish:     canPublish,
		CanParticipate: canParticipate,
		CreatedAt:      credit.CreatedAt.Unix(),
		UpdatedAt:      credit.UpdatedAt.Unix(),
	}, nil
}
