// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/userbasicservice"
	"activity-platform/common/errorx"
	ctxUtils "activity-platform/common/utils/context"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 修改用户信息
func NewUpdateUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *UpdateUserInfoLogic {
	return &UpdateUserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *UpdateUserInfoLogic) UpdateUserInfo(req *types.UpdateUserInfoReq) (resp *types.UpdateUserInfoResp, err error) {
	// 校验年龄
	if req.Age != 0 && (req.Age <= 0 || req.Age >= 200) {
		return nil, errorx.NewWithMessage(errorx.CodeInvalidParams, "年龄必须为正数并且小于两百岁")
	}

	// 校验并转换性别
	var genderInt int64
	if req.Gender != "" {
		if req.Gender == "男" || req.Gender == "1" {
			genderInt = 1
		} else if req.Gender == "女" || req.Gender == "2" {
			genderInt = 2
		} else {
			return nil, errorx.NewWithMessage(errorx.CodeInvalidParams, "性别只能是男或女")
		}
	}

	userId, err := ctxUtils.GetUserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	// 处理头像文件
	var avatarImgName string
	var avatarImgData []byte

	// 尝试获取文件，忽略错误（如果没有上传文件，err 会是 http.ErrMissingFile，此时不处理头像更新）
	file, header, err := l.r.FormFile("avatar_image")
	if err == nil {
		defer file.Close()
		avatarImgName = header.Filename
		avatarImgData, err = io.ReadAll(file)
		if err != nil {
			return nil, err
		}
	}

	// 调用 RPC
	rpcResp, err := l.svcCtx.UserBasicServiceRpc.UpdateUserInfo(l.ctx, &userbasicservice.UpdateUserInfoReq{
		UserId:        userId,
		Nickname:      req.Nickname,
		Introduce:     req.Introduction,
		Gender:        genderInt,
		AvatarImgName: avatarImgName,
		AvatarImgData: avatarImgData,
		Age:           req.Age,
		TagIds:        req.InterestTagIds,
	})
	if err != nil {
		return nil, err
	}

	return &types.UpdateUserInfoResp{
		UserInfo: types.UserInfo{
			UserId:            int64(rpcResp.UserId),
			Nickname:          rpcResp.Nickname,
			AvatarUrl:         rpcResp.AvatarUrl,
			Introduction:      rpcResp.Introduce,
			Gender:            strconv.FormatInt(rpcResp.Gender, 10),
			Age:               strconv.FormatInt(rpcResp.Age, 10),
			ActivitiesNum:     resp.ActivitiesNum,
			InitiateNum:       resp.InitiateNum,
			Credit:            resp.Credit,
			IsStudentVerified: resp.IsStudentVerified,
			InterestTags:     resp.InterestTags, // RPC response only returns TagIds, not full tag info
		},
	}, nil
}
