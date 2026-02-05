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

type DeleteActivityActionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteActivityActionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteActivityActionLogic {
	return &DeleteActivityActionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DeleteActivityAction 删除活动 - 正向操作
//
// DTM SAGA 分支的正向操作，负责软删除活动记录。
// 使用子事务屏障（barrier）解决以下问题：
//   - 幂等性：重复请求不会重复删除
//   - 悬挂：如果补偿操作先到达，正向操作会被跳过
//
// 删除策略：
//   - 软删除活动记录（设置 deleted_at）
//   - 保留关联的标签关系（由 User 服务的分支处理计数）
//
// 注意：此方法仅供 DTM Server 调用，不对外暴露
func (l *DeleteActivityActionLogic) DeleteActivityAction(in *activity.DeleteActivityActionReq) (*activity.DeleteActivityActionResp, error) {
	l.Infof("[DTM-Branch] DeleteActivityAction 开始: activity_id=%d, operator_id=%d, is_admin=%v",
		in.ActivityId, in.OperatorId, in.IsAdmin)

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

	// 3. 使用 barrier.CallWithDB 执行业务逻辑
	err = barrier.CallWithDB(sqlDB, func(tx *sql.Tx) error {
		now := time.Now()

		// 3.1 构建删除条件
		// 如果是管理员，可以删除任何活动
		// 如果不是管理员，只能删除自己创建的活动
		var result sql.Result
		var execErr error

		if in.IsAdmin {
			result, execErr = tx.ExecContext(l.ctx, `
				UPDATE activities
				SET deleted_at = ?
				WHERE id = ? AND deleted_at IS NULL
			`, now, in.ActivityId)
		} else {
			result, execErr = tx.ExecContext(l.ctx, `
				UPDATE activities
				SET deleted_at = ?
				WHERE id = ? AND organizer_id = ? AND deleted_at IS NULL
			`, now, in.ActivityId, in.OperatorId)
		}

		if execErr != nil {
			l.Errorf("[DTM-Branch] 软删除活动失败: %v", execErr)
			return execErr
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			// 活动不存在、已删除、或无权限
			// 根据业务需求决定是否报错
			// 这里选择记录日志但不报错，让事务继续
			l.Infof("[DTM-Branch] 活动删除无影响: activity_id=%d, operator_id=%d (可能不存在/已删除/无权限)",
				in.ActivityId, in.OperatorId)
			return nil
		}

		// 3.2 删除关联的标签关系
		_, err = tx.ExecContext(l.ctx, `
			DELETE FROM activity_tags WHERE activity_id = ?
		`, in.ActivityId)
		if err != nil {
			l.Errorf("[DTM-Branch] 删除标签关联失败: %v", err)
			return err
		}

		l.Infof("[DTM-Branch] 活动删除成功: activity_id=%d", in.ActivityId)
		return nil
	})

	// 4. 处理 barrier 执行结果
	if err != nil {
		l.Errorf("[DTM-Branch] DeleteActivityAction 失败: %v", err)
		return nil, status.Error(codes.Aborted, err.Error())
	}

	// 5. 返回成功响应
	return &activity.DeleteActivityActionResp{
		Success: true,
	}, nil
}
