package userbasicservicelogic

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"activity-platform/app/user/model"
	creditservicelogic "activity-platform/app/user/rpc/internal/logic/creditservice"
	qqemaillogic "activity-platform/app/user/rpc/internal/logic/qqemail"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"
	"activity-platform/common/utils/encrypt"
	"activity-platform/common/utils/jwt"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 用户注册
func (l *RegisterLogic) Register(in *pb.RegisterReq) (*pb.RegisterResponse, error) {
	// 1. 校验验证码 (复用 CheckQQEmailLogic 的逻辑)
	checkEmailLogic := qqemaillogic.NewCheckQQEmailLogic(l.ctx, l.svcCtx)
	_, err := checkEmailLogic.CheckQQEmail(&pb.CheckQQEmailReq{
		QqEmail: in.QqEmail,
		QqCode:  in.QqCode,
		Scene:   "register",
	})
	if err != nil {
		return nil, err
	}

	// 校验密码格式
	if !encrypt.ValidatePassword(in.Password) {
		return nil, errorx.NewWithMessage(errorx.CodePasswordInvalid, "密码长度必须为8-20个字符，且包含至少3种字符（大写字母、小写字母、数字、特殊字符）")
	}

	// 2. 检查邮箱是否已注册
	exists, err := l.svcCtx.UserModel.ExistsByQQEmail(l.ctx, in.QqEmail)
	if err != nil {
		l.Logger.Errorf("Check email existence failed: %v", err)
		return nil, errorx.ErrDBError(err)
	}
	if exists {
		return nil, errorx.New(errorx.CodeUserEmailAlreadyExists)
	}

	// 3. 创建用户
	newUser := &model.User{
		QQEmail:    in.QqEmail,
		Nickname:   in.Nickname,
		Password:   encrypt.EncryptPassword(in.Password),
		Status:     model.UserStatusNormal,
		Gender:     model.UserGenderUnknown, // 默认未知
		Age:        0,                       // 默认0
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}

	if err := l.svcCtx.UserModel.Create(l.ctx, newUser); err != nil {
		l.Logger.Errorf("Create user failed: %v", err)
		return nil, errorx.New(errorx.CodeUserRegisterFailed)
	}

	// 初始化信誉分
	initCreditLogic := creditservicelogic.NewInitCreditLogic(l.ctx, l.svcCtx)
	_, err = initCreditLogic.InitCredit(&pb.InitCreditReq{
		UserId: int64(newUser.UserID),
	})
	if err != nil {
		l.Logger.Errorf("Init credit score failed: %v, userId: %d", err, newUser.UserID)
		// 即使失败也不中断注册流程
	}

	// 获取信誉分
	var creditScore int64
	getCreditInfoLogic := creditservicelogic.NewGetCreditInfoLogic(l.ctx, l.svcCtx)
	creditInfo, err := getCreditInfoLogic.GetCreditInfo(&pb.GetCreditInfoReq{
		UserId: int64(newUser.UserID),
	})
	if err == nil {
		creditScore = creditInfo.Score
	} else {
		l.Logger.Errorf("Get credit info failed: %v, userId: %d", err, newUser.UserID)
	}

	// 4. 生成Token (自动登录)
	accessJwtId := uuid.New().String()
	refreshJwtId := uuid.New().String()

	shortToken, err := jwt.GenerateShortToken(newUser.UserID, jwt.RoleUser, jwt.AuthConfig{
		Secret: l.svcCtx.Config.JWT.AccessSecret,
		Expire: l.svcCtx.Config.JWT.AccessExpire,
	}, accessJwtId, refreshJwtId)
	if err != nil {
		l.Logger.Errorf("Generate short token failed: %v", err)
		return nil, errorx.New(errorx.CodeTokenGenerateFailed)
	}
	longToken, err := jwt.GenerateLongToken(newUser.UserID, jwt.RoleUser, jwt.AuthConfig{
		Secret: l.svcCtx.Config.JWT.RefreshSecret,
		Expire: l.svcCtx.Config.JWT.RefreshExpire,
	}, accessJwtId, refreshJwtId)
	if err != nil {
		l.Logger.Errorf("Generate long token failed: %v", err)
		return nil, errorx.New(errorx.CodeTokenGenerateFailed)
	}

	// 记录长token到redis
	// key: token:refresh:{refreshJwtId}  value: userId
	refreshTokenKey := fmt.Sprintf("token:refresh:%s", refreshJwtId)
	if err := l.svcCtx.Redis.Set(l.ctx, refreshTokenKey, newUser.UserID, time.Duration(l.svcCtx.Config.JWT.RefreshExpire)*time.Second).Err(); err != nil {
		l.Logger.Errorf("Set refresh token to redis failed: %v", err)
		// 不影响主流程，因为已经注册成功且返回了token
	}

	// 5. 返回响应
	return &pb.RegisterResponse{
		AccessToken:  shortToken.Token,
		RefreshToken: longToken.Token,
		UserInfo: &pb.UserInfo{
			UserId:        uint64(newUser.UserID),
			Nickname:      newUser.Nickname,
			AvatarUrl:     "",
			Introduction:  newUser.Introduction,
			Gender:        "未知",
			Age:           strconv.FormatInt(newUser.Age, 10),
			ActivitiesNum: 0,
			InitiateNum:   0,
			Credit:        creditScore,
			QqEmail:       newUser.QQEmail,
		},
	}, nil
}
