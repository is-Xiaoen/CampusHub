package userbasicservicelogic

import (
	"context"

	creditservicelogic "activity-platform/app/user/rpc/internal/logic/creditservice"
	tagservicelogic "activity-platform/app/user/rpc/internal/logic/tagservice"
	uploadtoqiniulogic "activity-platform/app/user/rpc/internal/logic/uploadtoqiniu"
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
		// 使用统一的 UploadAvatar RPC 处理删除旧图、上传新图与DB更新
		uploadResp, err := uploadtoqiniulogic.NewUploadAvatarLogic(l.ctx, l.svcCtx).UploadAvatar(&pb.UploadAvatarReq{
			UserId:   in.UserId,
			FileName: in.AvatarImgName,
			FileData: in.AvatarImgData,
		})
		if err != nil {
			return nil, err
		}
		// 同步内存对象的头像URL（DB已更新）
		user.AvatarURL = uploadResp.AvatarUrl
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

	// 5. 获取信誉分 (CreditService)
	var creditScore int64
	getCreditInfoLogic := creditservicelogic.NewGetCreditInfoLogic(l.ctx, l.svcCtx)
	creditInfo, errCredit := getCreditInfoLogic.GetCreditInfo(&pb.GetCreditInfoReq{
		UserId: int64(user.UserID),
	})
	if errCredit == nil {
		creditScore = creditInfo.Score
	} else {
		l.Logger.Errorf("Get credit info failed: %v, userId: %d", errCredit, user.UserID)
	}

	return &pb.UpdateUserInfoResponse{
		UserId:    user.UserID,
		Nickname:  user.Nickname,
		Introduce: user.Introduction,
		Gender:    user.Gender,
		AvatarUrl: user.AvatarURL,
		Age:       user.Age,
		TagIds:    in.TagIds,
		QqEmail:   user.QQEmail,
		Credit:    creditScore,
	}, nil
}
