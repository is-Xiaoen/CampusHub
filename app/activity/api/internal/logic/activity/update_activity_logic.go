package activity

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"
	"activity-platform/app/activity/rpc/activityservice"
	"activity-platform/common/ctxdata"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新活动
func NewUpdateActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateActivityLogic {
	return &UpdateActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateActivityLogic) UpdateActivity(req *types.UpdateActivityReq) (resp *types.UpdateActivityResp, err error) {
	// 1. 获取当前用户 ID
	userID := ctxdata.GetUserIDFromCtx(l.ctx)
	if userID <= 0 {
		return nil, errorx.ErrUnauthorized()
	}

	// 2. 参数校验
	if req.Id <= 0 {
		return nil, errorx.ErrInvalidParams("活动ID无效")
	}
	if req.Version <= 0 {
		return nil, errorx.ErrInvalidParams("版本号无效")
	}

	// 2.1 可选字段值校验（API 层快速失败，避免无效 RPC 调用）
	if req.Title != nil {
		titleLen := len([]rune(*req.Title))
		if titleLen < 2 || titleLen > 100 {
			return nil, errorx.ErrInvalidParams("标题长度需在2-100字符之间")
		}
	}
	if req.CoverUrl != nil && *req.CoverUrl == "" {
		return nil, errorx.ErrInvalidParams("封面URL不能为空")
	}
	if req.Location != nil && *req.Location == "" {
		return nil, errorx.ErrInvalidParams("活动地点不能为空")
	}
	if len(req.TagIds) > 5 {
		return nil, errorx.ErrInvalidParams("最多选择5个标签")
	}

	// 3. 构建 RPC 请求（处理可选字段）
	rpcReq := &activityservice.UpdateActivityReq{
		Id:         req.Id,
		Version:    req.Version,
		OperatorId: userID,
		TagIds:     req.TagIds,
		UpdateTags: req.UpdateTags,
	}

	// 可选字段映射（只有非 nil 才设置）
	if req.Title != nil {
		rpcReq.Title = req.Title
	}
	if req.CoverUrl != nil {
		rpcReq.CoverUrl = req.CoverUrl
	}
	if req.CoverType != nil {
		rpcReq.CoverType = req.CoverType
	}
	if req.Content != nil {
		rpcReq.Content = req.Content
	}
	if req.CategoryId != nil {
		rpcReq.CategoryId = req.CategoryId
	}
	if req.ContactPhone != nil {
		rpcReq.ContactPhone = req.ContactPhone
	}
	if req.RegisterStartTime != nil {
		rpcReq.RegisterStartTime = req.RegisterStartTime
	}
	if req.RegisterEndTime != nil {
		rpcReq.RegisterEndTime = req.RegisterEndTime
	}
	if req.ActivityStartTime != nil {
		rpcReq.ActivityStartTime = req.ActivityStartTime
	}
	if req.ActivityEndTime != nil {
		rpcReq.ActivityEndTime = req.ActivityEndTime
	}
	if req.Location != nil {
		rpcReq.Location = req.Location
	}
	if req.AddressDetail != nil {
		rpcReq.AddressDetail = req.AddressDetail
	}
	if req.Longitude != nil {
		rpcReq.Longitude = req.Longitude
	}
	if req.Latitude != nil {
		rpcReq.Latitude = req.Latitude
	}
	if req.MaxParticipants != nil {
		rpcReq.MaxParticipants = req.MaxParticipants
	}
	if req.RequireApproval != nil {
		rpcReq.RequireApproval = req.RequireApproval
	}
	if req.RequireStudentVerify != nil {
		rpcReq.RequireStudentVerify = req.RequireStudentVerify
	}
	if req.MinCreditScore != nil {
		rpcReq.MinCreditScore = req.MinCreditScore
	}

	// 4. 调用 RPC 更新活动
	rpcResp, err := l.svcCtx.ActivityRpc.UpdateActivity(l.ctx, rpcReq)
	if err != nil {
		l.Errorf("RPC UpdateActivity failed: id=%d, userID=%d, err=%v", req.Id, userID, err)
		return nil, errorx.FromError(err)
	}

	// 5. 返回响应
	return &types.UpdateActivityResp{
		Status:     rpcResp.Status,
		NewVersion: rpcResp.NewVersion,
	}, nil
}
