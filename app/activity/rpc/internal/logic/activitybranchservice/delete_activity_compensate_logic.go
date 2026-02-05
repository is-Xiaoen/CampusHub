package activitybranchservicelogic

import (
	"context"
	"database/sql"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/dtm-labs/client/dtmgrpc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DeleteActivityCompensateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteActivityCompensateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteActivityCompensateLogic {
	return &DeleteActivityCompensateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DeleteActivityCompensate 删除活动 - 补偿操作（回滚）
//
// DTM SAGA 分支的补偿操作，用于恢复已删除的活动。
// 使用子事务屏障（barrier）解决以下问题：
//   - 幂等性：重复补偿不会重复执行
//   - 空补偿：如果正向操作未执行，补偿操作会被跳过
//
// 补偿策略：
//   - 恢复软删除的活动记录（清除 deleted_at）
//   - 恢复关联的标签关系（根据传入的 tag_ids）
//
// 注意：此方法仅供 DTM Server 调用，不对外暴露
func (l *DeleteActivityCompensateLogic) DeleteActivityCompensate(in *activity.DeleteActivityCompensateReq) (*activity.DeleteActivityCompensateResp, error) {
	l.Infof("[DTM-Branch] DeleteActivityCompensate 开始: activity_id=%d, tag_ids=%v",
		in.ActivityId, in.TagIds)

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
	err = barrier.CallWithDB(sqlDB, func(tx *sql.Tx) error {
		// 3.1 恢复活动记录（清除 deleted_at）
		result, err := tx.ExecContext(l.ctx, `
			UPDATE activities
			SET deleted_at = NULL
			WHERE id = ? AND deleted_at IS NOT NULL
		`, in.ActivityId)
		if err != nil {
			l.Errorf("[DTM-Branch] 恢复活动失败: %v", err)
			return err
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			// 活动不存在或未被删除，这是正常情况（可能是空补偿）
			l.Infof("[DTM-Branch] 活动未被删除或不存在: activity_id=%d", in.ActivityId)
			return nil
		}

		// 3.2 恢复标签关联关系
		if len(in.TagIds) > 0 {
			// 使用批量插入恢复标签关联
			stmt, err := tx.PrepareContext(l.ctx, `
				INSERT IGNORE INTO activity_tags (activity_id, tag_id, created_at)
				VALUES (?, ?, UNIX_TIMESTAMP())
			`)
			if err != nil {
				l.Errorf("[DTM-Branch] 准备标签插入语句失败: %v", err)
				return err
			}
			defer stmt.Close()

			for _, tagID := range in.TagIds {
				_, err = stmt.ExecContext(l.ctx, in.ActivityId, tagID)
				if err != nil {
					l.Errorf("[DTM-Branch] 恢复标签关联失败: tag_id=%d, err=%v", tagID, err)
					return err
				}
			}
			l.Infof("[DTM-Branch] 恢复了 %d 个标签关联", len(in.TagIds))
		}

		l.Infof("[DTM-Branch] 活动补偿成功（已恢复）: activity_id=%d", in.ActivityId)
		return nil
	})

	// 4. 处理 barrier 执行结果
	if err != nil {
		l.Errorf("[DTM-Branch] DeleteActivityCompensate 失败: %v", err)
		return nil, status.Error(codes.Aborted, err.Error())
	}

	// 5. 返回成功响应
	return &activity.DeleteActivityCompensateResp{
		Success: true,
	}, nil
}
