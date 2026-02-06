package userbasicservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/logic/tagservice"
	"activity-platform/app/user/rpc/internal/logic/uploadtoqiniu"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserInfoLogic {
	return &UpdateUserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 修改用户信息
func (l *UpdateUserInfoLogic) UpdateUserInfo(in *pb.UpdateUserInfoReq) (*pb.UpdateUserInfoResponse, error) {
	// 1. 查询当前用户
	user, err := l.svcCtx.UserModel.FindByUserID(l.ctx, in.UserId)
	if err != nil {
		l.Logger.Errorf("FindByUserID error: %v, userId: %d", err, in.UserId)
		return nil, errorx.ErrDBError(err)
	}
	if user == nil {
		return nil, errorx.New(errorx.CodeUserNotFound)
	}

	// 2. 处理头像上传与更新
	// 只有当提供了新的头像数据时才处理
	if len(in.AvatarImgData) > 0 {
		uploadLogic := uploadtoqiniulogic.NewUploadFileLogic(l.ctx, l.svcCtx)
		uploadResp, err := uploadLogic.UploadFile(&pb.UploadFileReq{
			FileName: in.AvatarImgName,
			FileData: in.AvatarImgData,
		})
		if err != nil {
			return nil, err // UploadFileLogic 已经返回了 status error
		}

		// 如果原用户有头像，且不是默认头像（如果系统有默认头像逻辑），则删除旧头像
		// 这里假设只要有 URL 就尝试删除
		if user.AvatarURL != "" {
			deleteLogic := uploadtoqiniulogic.NewDeleteFileLogic(l.ctx, l.svcCtx)
			// 删除旧头像失败不阻断流程，仅记录日志
			_, err := deleteLogic.DeleteFile(&pb.DeleteFileReq{
				FileUrl: user.AvatarURL,
			})
			if err != nil {
				l.Logger.Errorf("Failed to delete old avatar: %s, error: %v", user.AvatarURL, err)
			}
		}

		// 更新为新头像 URL
		user.AvatarURL = uploadResp.FileUrl
	}

	// 3. 更新基本信息
	// 这里直接覆盖，前端需要确保回填所有字段
	user.Nickname = in.Nickname
	user.Introduction = in.Introduce
	user.Gender = in.Gender
	user.Age = in.Age

	err = l.svcCtx.UserModel.Update(l.ctx, user)
	if err != nil {
		l.Logger.Errorf("Update user error: %v, userId: %d", err, in.UserId)
		return nil, errorx.New(errorx.CodeUserUpdateFailed)
	}

	// 4. 更新标签
	// 调用 UpdateUserTagLogic
	tagLogic := tagservicelogic.NewUpdateUserTagLogic(l.ctx, l.svcCtx)
	_, err = tagLogic.UpdateUserTag(&pb.UpdateUserTagReq{
		UserId: in.UserId,
		Ids:    in.TagIds,
	})
	if err != nil {
		return nil, err
	}

	return &pb.UpdateUserInfoResponse{
		UserId:    user.UserID,
		Nickname:  user.Nickname,
		Introduce: user.Introduction,
		Gender:    user.Gender,
		AvatarUrl: user.AvatarURL,
		Age:       user.Age,
		TagIds:    in.TagIds,
	}, nil
}
