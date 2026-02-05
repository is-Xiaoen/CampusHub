// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/tagservice"
	ctxUtils "activity-platform/common/utils/context"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateInterestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 修改兴趣
func NewUpdateInterestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateInterestLogic {
	return &UpdateInterestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateInterestLogic) UpdateInterest(req *types.UpdateInterestReq) (resp *types.UpdateInterestResp, err error) {
	userId, err := ctxUtils.GetUserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	rpcResp, err := l.svcCtx.TagServiceRpc.UpdateUserTag(l.ctx, &tagservice.UpdateUserTagReq{
		UserId: userId,
		Ids:    req.InterestTagIds,
	})
	if err != nil {
		return nil, err
	}

	var interestTags []types.InterestTag
	for _, tag := range rpcResp.Tags {
		interestTags = append(interestTags, types.InterestTag{
			Id:       int64(tag.Id),
			TagName:  tag.Name,
			TagColor: tag.Color,
			TagIcon:  tag.Icon,
			TagDesc:  tag.Description,
		})
	}

	return &types.UpdateInterestResp{
		UserId:       userId,
		InterestTags: interestTags,
	}, nil
}
