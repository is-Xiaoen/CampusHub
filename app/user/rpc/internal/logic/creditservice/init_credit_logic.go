/**
 * @projectName: CampusHub
 * @package: creditservicelogic
 * @className: InitCreditLogic
 * @author: lijunqi
 * @description: 初始化信用分逻辑层（含缓存写入）
 * @date: 2026-01-30
 * @version: 1.0
 */

package creditservicelogic

import (
	"context"

	"activity-platform/app/user/model"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/constants"
	"activity-platform/common/errorx"
	"activity-platform/common/utils/idgen"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// InitCreditLogic 初始化信用分逻辑处理器
type InitCreditLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewInitCreditLogic 创建初始化信用分逻辑实例
func NewInitCreditLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InitCreditLogic {
	return &InitCreditLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// InitCredit 初始化信用分
// 业务逻辑:
//   - 用户注册成功后调用，初始化信用分为100分（Lv4社区之星）
//   - 幂等处理：如果已存在信用记录，返回已初始化错误
//   - 写入 MySQL 后同步写入 Redis 缓存
func (l *InitCreditLogic) InitCredit(in *pb.InitCreditReq) (*pb.InitCreditResp, error) {
	// 1. 参数校验
	if in.UserId <= 0 {
		l.Errorf("InitCredit 参数错误: userId=%d", in.UserId)
		return nil, errorx.ErrInvalidParams("用户ID无效")
	}

	// 2. 检查是否已初始化（幂等）
	exists, err := l.svcCtx.UserCreditModel.ExistsByUserID(l.ctx, in.UserId)
	if err != nil {
		l.Errorf("InitCredit 检查信用记录失败: userId=%d, err=%v", in.UserId, err)
		return nil, errorx.ErrDBError(err)
	}
	if exists {
		l.Infof("InitCredit 信用分已初始化: userId=%d", in.UserId)
		return nil, errorx.ErrCreditAlreadyInit()
	}

	// 3. 初始化信用分
	initScore := constants.CreditScoreInit
	initLevel := constants.CalculateCreditLevel(initScore)

	// 构建信用记录（ID由数据库自增生成）
	credit := &model.UserCredit{
		UserID: in.UserId,
		Score:  initScore,
		Level:  initLevel,
	}

	// 4. 使用事务同时创建信用记录和初始化日志
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		// 4.1 创建信用记录
		if err := tx.Create(credit).Error; err != nil {
			return err
		}

		// 4.2 创建初始化日志（幂等键: init:{userId}，ID由数据库自增）
		sourceID := idgen.GenInitSourceID(in.UserId)
		creditLog := &model.CreditLog{
			UserID:      in.UserId,
			ChangeType:  model.CreditChangeTypeAdd,
			SourceID:    sourceID,
			BeforeScore: 0,
			AfterScore:  initScore,
			Delta:       initScore,
			Reason:      "新用户注册，初始化信用分",
		}
		return tx.Create(creditLog).Error
	})

	if err != nil {
		l.Errorf("InitCredit 初始化信用分失败: userId=%d, err=%v", in.UserId, err)
		return nil, errorx.ErrDBError(err)
	}

	// 5. 写入 Redis 缓存（DB成功后同步写入，确保新用户首次查询命中缓存）
	if err := l.svcCtx.CreditCache.Set(l.ctx, in.UserId, initScore, initLevel); err != nil {
		l.Errorf("InitCredit 写入缓存失败: userId=%d, err=%v", in.UserId, err)
		// 缓存写入失败不影响主流程，下次读取会回填
	}

	l.Infof("InitCredit 初始化成功: userId=%d, score=%d, level=%d", in.UserId, initScore, initLevel)

	return &pb.InitCreditResp{
		Success: true,
		Score:   int64(initScore),
		Level:   int32(initLevel),
	}, nil
}
