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

type CreateActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建活动
func NewCreateActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateActivityLogic {
	return &CreateActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateActivityLogic) CreateActivity(req *types.CreateActivityReq) (resp *types.CreateActivityResp, err error) {
	// 1. 获取当前用户 ID（从 JWT 解析）
	userID := ctxdata.GetUserIDFromCtx(l.ctx)
	if userID <= 0 {
		return nil, errorx.ErrUnauthorized()
	}

	// 2. 参数校验
	if err := l.validateParams(req); err != nil {
		return nil, err
	}

	// 3. 调用 RPC 创建活动
	// OrganizerName/Avatar 传空，RPC 层通过 UserBasicRpc.GetUserInfo 自动填充
	rpcResp, err := l.svcCtx.ActivityRpc.CreateActivity(l.ctx, &activityservice.CreateActivityReq{
		Title:                req.Title,
		CoverUrl:             req.CoverUrl,
		CoverType:            req.CoverType,
		Content:              req.Content,
		CategoryId:           req.CategoryId,
		ContactPhone:         req.ContactPhone,
		RegisterStartTime:    req.RegisterStartTime,
		RegisterEndTime:      req.RegisterEndTime,
		ActivityStartTime:    req.ActivityStartTime,
		ActivityEndTime:      req.ActivityEndTime,
		Location:             req.Location,
		AddressDetail:        req.AddressDetail,
		Longitude:            req.Longitude,
		Latitude:             req.Latitude,
		MaxParticipants:      req.MaxParticipants,
		RequireApproval:      req.RequireApproval,
		RequireStudentVerify: req.RequireStudentVerify,
		MinCreditScore:       req.MinCreditScore,
		TagIds:               req.TagIds,
		IsDraft:              req.IsDraft,
		OrganizerId:          userID,
		OrganizerName:        "", // RPC 层通过 UserBasicRpc 自动获取
		OrganizerAvatar:      "", // RPC 层通过 UserBasicRpc 自动获取
	})
	if err != nil {
		l.Errorf("RPC CreateActivity failed: userID=%d, title=%s, err=%v", userID, req.Title, err)
		return nil, errorx.FromError(err)
	}

	// 4. 返回响应
	return &types.CreateActivityResp{
		Id:     rpcResp.Id,
		Status: rpcResp.Status,
	}, nil
}

// validateParams API 层参数校验（快速失败，避免不必要的 RPC 调用）
func (l *CreateActivityLogic) validateParams(req *types.CreateActivityReq) error {
	// 标题长度校验
	titleLen := len([]rune(req.Title))
	if titleLen < 2 {
		return errorx.ErrInvalidParams("标题至少2个字符")
	}
	if titleLen > 100 {
		return errorx.ErrInvalidParams("标题不能超过100个字符")
	}

	// 封面校验
	if req.CoverUrl == "" {
		return errorx.ErrInvalidParams("请上传活动封面")
	}

	// 分类校验
	if req.CategoryId <= 0 {
		return errorx.ErrInvalidParams("请选择活动分类")
	}

	// 地点校验
	if req.Location == "" {
		return errorx.ErrInvalidParams("请填写活动地点")
	}

	// 时间校验（基础）
	if req.RegisterStartTime <= 0 || req.RegisterEndTime <= 0 ||
		req.ActivityStartTime <= 0 || req.ActivityEndTime <= 0 {
		return errorx.ErrInvalidParams("请填写完整的时间信息")
	}

	// 标签数量校验
	if len(req.TagIds) > 5 {
		return errorx.ErrInvalidParams("最多选择5个标签")
	}

	return nil
}
