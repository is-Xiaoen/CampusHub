/**
 * @projectName: CampusHub
 * @package: creditservicelogic
 * @className: CanPublishLogic
 * @author: lijunqi
 * @description: 校验发布资格逻辑层
 * @date: 2026-01-30
 * @version: 1.0
 */

package creditservicelogic

import (
	"context"
	"fmt"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/constants"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// CanPublishLogic 校验发布资格逻辑处理器
type CanPublishLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewCanPublishLogic 创建校验发布资格逻辑实例
func NewCanPublishLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CanPublishLogic {
	return &CanPublishLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CanPublish 校验是否允许发布活动
// 业务逻辑:
//   - score >= 90: 允许发布（Lv3优秀用户、Lv4社区之星）
//   - score < 90: 禁止发布
func (l *CanPublishLogic) CanPublish(in *pb.CanPublishReq) (*pb.CanPublishResp, error) {
	// 1. 参数校验
	if in.UserId <= 0 {
		l.Errorf("CanPublish 参数错误: userId=%d", in.UserId)
		return nil, errorx.ErrInvalidParams("用户ID无效")
	}

	// 2. 查询信用记录
	credit, err := l.svcCtx.UserCreditModel.FindByUserID(l.ctx, in.UserId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			l.Infof("CanPublish 信用记录不存在: userId=%d", in.UserId)
			return nil, errorx.ErrCreditNotFound()
		}
		l.Errorf("CanPublish 查询信用记录失败: userId=%d, err=%v", in.UserId, err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 根据信用分判断是否允许发布
	score := credit.Score
	level := credit.Level

	// 3.1 信用分不足（score < 90）：禁止发布
	if score < constants.CreditThresholdPublish {
		l.Infof("CanPublish 信用分不足禁止发布: userId=%d, score=%d, threshold=%d",
			in.UserId, score, constants.CreditThresholdPublish)
		return &pb.CanPublishResp{
			Allowed: false,
			Reason:  fmt.Sprintf("信用分不足%d分（当前%d分），暂时无法发布活动，请先通过参与活动积累信用", constants.CreditThresholdPublish, score),
			Score:   int64(score),
			Level:   int32(level),
		}, nil
	}

	// 3.2 信用分充足（score >= 90）：允许发布
	l.Infof("CanPublish 允许发布: userId=%d, score=%d, level=%d", in.UserId, score, level)

	return &pb.CanPublishResp{
		Allowed: true,
		Reason:  "",
		Score:   int64(score),
		Level:   int32(level),
	}, nil
}
