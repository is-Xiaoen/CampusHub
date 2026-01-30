/**
 * @projectName: CampusHub
 * @package: creditservicelogic
 * @className: UpdateScoreLogic
 * @author: lijunqi
 * @description: 变更信用分逻辑层
 * @date: 2026-01-30
 * @version: 1.0
 */

package creditservicelogic

import (
	"context"
	"strings"

	"activity-platform/app/user/model"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/constants"
	"activity-platform/common/errorx"

	"github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// UpdateScoreLogic 变更信用分逻辑处理器
type UpdateScoreLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewUpdateScoreLogic 创建变更信用分逻辑实例
func NewUpdateScoreLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateScoreLogic {
	return &UpdateScoreLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UpdateScore 变更信用分
// 业务逻辑:
//   - 签到、爽约、活动结束等事件触发信用分变更
//   - 幂等保证: 通过数据库唯一索引 uk_source_id 实现
//   - 分数范围: 0-100
func (l *UpdateScoreLogic) UpdateScore(in *pb.UpdateScoreReq) (*pb.UpdateScoreResp, error) {
	// 1. 参数校验
	if err := l.validateRequest(in); err != nil {
		return nil, err
	}

	// 2. 查询当前信用记录
	credit, err := l.getCreditInfo(in.UserId)
	if err != nil {
		return nil, err
	}

	// 3. 计算分数变动
	result := l.calculateScoreChange(credit, in)
	if result.skipUpdate {
		return result.toResponse(), nil
	}

	// 4. 执行事务更新（幂等由数据库唯一索引保证）
	if err := l.executeUpdate(in, result); err != nil {
		return nil, err
	}

	return result.toResponse(), nil
}

// ==================== 私有辅助方法 ====================

// validateRequest 参数校验
func (l *UpdateScoreLogic) validateRequest(in *pb.UpdateScoreReq) error {
	if in.UserId <= 0 {
		l.Errorf("UpdateScore 参数错误: userId=%d", in.UserId)
		return errorx.ErrInvalidParams("用户ID无效")
	}
	if in.SourceId == "" {
		l.Errorf("UpdateScore 参数错误: sourceId为空")
		return errorx.ErrInvalidParams("来源ID不能为空")
	}
	if in.ChangeType <= 0 {
		l.Errorf("UpdateScore 参数错误: changeType=%d", in.ChangeType)
		return errorx.ErrInvalidParams("变更类型无效")
	}
	return nil
}

// getCreditInfo 获取用户信用信息
func (l *UpdateScoreLogic) getCreditInfo(userID int64) (*model.UserCredit, error) {
	credit, err := l.svcCtx.UserCreditModel.FindByUserID(l.ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			l.Infof("UpdateScore 信用记录不存在: userId=%d", userID)
			return nil, errorx.ErrCreditNotFound()
		}
		l.Errorf("UpdateScore 查询信用记录失败: userId=%d, err=%v", userID, err)
		return nil, errorx.ErrDBError(err)
	}
	return credit, nil
}

// scoreChangeResult 分数变更计算结果
type scoreChangeResult struct {
	beforeScore int
	afterScore  int
	actualDelta int
	newLevel    int8
	changeType  int8
	skipUpdate  bool
}

// toResponse 转换为响应
func (r *scoreChangeResult) toResponse() *pb.UpdateScoreResp {
	return &pb.UpdateScoreResp{
		Success:     true,
		BeforeScore: int64(r.beforeScore),
		AfterScore:  int64(r.afterScore),
		Delta:       int64(r.actualDelta),
		NewLevel:    int32(r.newLevel),
	}
}

// calculateScoreChange 计算分数变动
func (l *UpdateScoreLogic) calculateScoreChange(credit *model.UserCredit, in *pb.UpdateScoreReq) *scoreChangeResult {
	result := &scoreChangeResult{
		beforeScore: credit.Score,
	}

	// 获取变动值
	delta := constants.GetCreditDelta(in.ChangeType, in.AdminDelta)

	// delta=0 且不是提前取消，无需处理
	if delta == 0 && in.ChangeType != constants.CreditChangeTypeCancelEarly {
		l.Infof("UpdateScore 分数变动为0，无需处理: userId=%d, changeType=%d", in.UserId, in.ChangeType)
		result.afterScore = credit.Score
		result.newLevel = credit.Level
		result.skipUpdate = true
		return result
	}

	// 计算新分数（限制在 0-100 范围内）
	result.afterScore = l.clampScore(credit.Score + delta)
	result.actualDelta = result.afterScore - result.beforeScore
	result.newLevel = constants.CalculateCreditLevel(result.afterScore)
	result.changeType = l.determineChangeType(result.actualDelta)

	return result
}

// clampScore 限制分数在有效范围内
func (l *UpdateScoreLogic) clampScore(score int) int {
	if score < constants.CreditScoreMin {
		return constants.CreditScoreMin
	}
	if score > constants.CreditScoreMax {
		return constants.CreditScoreMax
	}
	return score
}

// determineChangeType 确定变更类型（加分/扣分）
func (l *UpdateScoreLogic) determineChangeType(delta int) int8 {
	if delta >= 0 {
		return model.CreditChangeTypeAdd
	}
	return model.CreditChangeTypeDeduct
}

// executeUpdate 执行事务更新
// 幂等保证：先插入日志（利用唯一索引），成功后再更新分数
func (l *UpdateScoreLogic) executeUpdate(in *pb.UpdateScoreReq, result *scoreChangeResult) error {
	err := l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 先插入日志（利用唯一索引 uk_source_id 保证幂等）
		if err := l.createCreditLog(tx, in, result); err != nil {
			// 检查是否为唯一索引冲突（幂等场景）
			if l.isDuplicateKeyError(err) {
				l.Infof("UpdateScore 幂等拦截（唯一索引）: sourceId=%s", in.SourceId)
				return errorx.ErrCreditSourceDup()
			}
			return err
		}

		// 2. 日志插入成功，更新信用分
		return l.updateCreditScore(tx, in.UserId, result)
	})

	if err != nil {
		// 幂等错误不算失败
		if errorx.IsCodeError(err) && errorx.GetCodeError(err).Code == errorx.CodeCreditSourceDup {
			return err
		}
		l.Errorf("UpdateScore 更新信用分失败: userId=%d, err=%v", in.UserId, err)
		return errorx.ErrDBError(err)
	}
	return nil
}

// isDuplicateKeyError 判断是否为唯一索引冲突错误
//
// MySQL 唯一索引冲突时会返回错误码 1062，错误信息格式如下：
//   - Error 1062: Duplicate entry 'xxx' for key 'uk_source_id'
//
// 常见 MySQL 错误码说明：
//   - 1062 (ER_DUP_ENTRY): 唯一索引/主键冲突，插入或更新的值已存在
//   - 1452 (ER_NO_REFERENCED_ROW_2): 外键约束失败，引用的记录不存在
//   - 1451 (ER_ROW_IS_REFERENCED_2): 外键约束失败，记录被其他表引用
//   - 1213 (ER_LOCK_DEADLOCK): 死锁，事务被回滚
//   - 1205 (ER_LOCK_WAIT_TIMEOUT): 锁等待超时
//
// 本函数用于幂等场景：当 source_id 唯一索引冲突时，说明该操作已处理过
func (l *UpdateScoreLogic) isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	// 方式1: 类型断言检查 MySQL 错误码（推荐，精确匹配）
	// mysql.MySQLError 是 go-sql-driver/mysql 驱动定义的错误类型
	// Number 字段为 MySQL 服务端返回的错误码
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		// 1062 = ER_DUP_ENTRY，唯一索引冲突
		return mysqlErr.Number == 1062
	}

	// 方式2: 字符串匹配（兼容其他数据库驱动或包装后的错误）
	// "Duplicate entry" - MySQL 原始错误信息
	// "duplicate key" - PostgreSQL/通用错误信息
	return strings.Contains(err.Error(), "Duplicate entry") ||
		strings.Contains(err.Error(), "duplicate key")
}

// updateCreditScore 更新信用分数
func (l *UpdateScoreLogic) updateCreditScore(tx *gorm.DB, userID int64, result *scoreChangeResult) error {
	return tx.Model(&model.UserCredit{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"score": result.afterScore,
			"level": result.newLevel,
		}).Error
}

// createCreditLog 创建信用变更日志
func (l *UpdateScoreLogic) createCreditLog(tx *gorm.DB, in *pb.UpdateScoreReq, result *scoreChangeResult) error {
	creditLog := &model.CreditLog{
		UserID:      in.UserId,
		ChangeType:  result.changeType,
		SourceID:    in.SourceId,
		BeforeScore: result.beforeScore,
		AfterScore:  result.afterScore,
		Delta:       result.actualDelta,
		Reason:      in.Reason,
	}
	return tx.Create(creditLog).Error
}
