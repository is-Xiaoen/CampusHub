package userbasicservicelogic

import (
	"context"

	creditservicelogic "activity-platform/app/user/rpc/internal/logic/creditservice"
	tagservicelogic "activity-platform/app/user/rpc/internal/logic/tagservice"
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

	// 2. 处理头像ID：使用 GetSysImage 查询URL，并更新 avatar_id
	var avatarURL string
	if in.AvatarId > 0 {
		imgResp, err := NewGetSysImageLogic(l.ctx, l.svcCtx).GetSysImage(&pb.GetSysImageReq{
			UserId:  in.UserId,
			ImageId: in.AvatarId,
		})
		if err != nil {
			return nil, err
		}
		// 不再保存 AvatarURL 到数据库，仅用于本次响应
		avatarURL = imgResp.Url
		// 更新 AvatarID（模型需包含该字段，数据库需存在 avatar_id）
		user.AvatarID = in.AvatarId
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
		AvatarUrl: avatarURL,
		Age:       user.Age,
		TagIds:    in.TagIds,
		QqEmail:   user.QQEmail,
		Credit:    creditScore,
	}, nil
}
