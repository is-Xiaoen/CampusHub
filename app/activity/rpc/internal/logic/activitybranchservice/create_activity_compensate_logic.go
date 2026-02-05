package activitybranchservicelogic

import (
	"context"
	"database/sql"
	"time"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/dtm-labs/client/dtmgrpc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CreateActivityCompensateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateActivityCompensateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateActivityCompensateLogic {
	return &CreateActivityCompensateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CreateActivityCompensate 创建活动 - 补偿操作（回滚）
//
// DTM SAGA 分支的补偿操作，用于回滚已创建的活动。
// 使用子事务屏障（barrier）解决以下问题：
//   - 幂等性：重复补偿不会重复执行
//   - 空补偿：如果正向操作未执行，补偿操作会被跳过
//
// 补偿策略：
//   - 软删除活动记录（设置 deleted_at）
//   - 同时删除关联的标签关系
//
// 注意：此方法仅供 DTM Server 调用，不对外暴露
func (l *CreateActivityCompensateLogic) CreateActivityCompensate(in *activity.CreateActivityCompensateReq) (*activity.CreateActivityCompensateResp, error) {
	l.Infof("[DTM-Branch] CreateActivityCompensate 开始: activity_id=%d", in.ActivityId)

	// 1. 从 gRPC context 获取 DTM 事务信息
	barrier, err := dtmgrpc.BarrierFromGrpc(l.ctx)
	if err != nil {
		l.Errorf("[DTM-Branch] 获取 barrier 失败: %v", err)
		return nil, status.Error(codes.Internal, "获取事务屏障失败")
	}

	// 2. 获取原生 SQL DB
	sqlDB, err := l.svcCtx.DB.DB()
	if err != nil {
		l.Errorf("[DTM-Branch] 获取 SQL DB 失败: %v", err)
		return nil, status.Error(codes.Internal, "获取数据库连接失败")
	}

	// 3. 使用 barrier.CallWithDB 执行补偿逻辑
	// barrier 会自动处理幂等和空补偿问题
	err = barrier.CallWithDB(sqlDB, func(tx *sql.Tx) error {
		now := time.Now()

		// 3.1 软删除活动记录
		result, err := tx.ExecContext(l.ctx, `
			UPDATE activities
			SET deleted_at = ?
			WHERE id = ? AND deleted_at IS NULL
		`, now, in.ActivityId)
		if err != nil {
			l.Errorf("[DTM-Branch] 软删除活动失败: %v", err)
			return err
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			// 活动不存在或已删除，这是正常情况（可能是空补偿或重复补偿）
			l.Infof("[DTM-Branch] 活动不存在或已删除: activity_id=%d", in.ActivityId)
			return nil
		}

		// 3.2 删除关联的标签关系（如果有）
		_, err = tx.ExecContext(l.ctx, `
			DELETE FROM activity_tags WHERE activity_id = ?
		`, in.ActivityId)
		if err != nil {
			l.Errorf("[DTM-Branch] 删除标签关联失败: %v", err)
			return err
		}

		l.Infof("[DTM-Branch] 活动补偿成功（已软删除）: activity_id=%d", in.ActivityId)
		return nil
	})

	// 4. 处理 barrier 执行结果
	if err != nil {
		l.Errorf("[DTM-Branch] CreateActivityCompensate 失败: %v", err)
		return nil, status.Error(codes.Aborted, err.Error())
	}

	// 5. 返回成功响应
	return &activity.CreateActivityCompensateResp{
		Success: true,
	}, nil
}
