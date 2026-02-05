package tagbranchservicelogic

import (
	"context"
	"database/sql"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/dtm-labs/client/dtmgrpc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DecrTagUsageCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDecrTagUsageCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DecrTagUsageCountLogic {
	return &DecrTagUsageCountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DecrTagUsageCount 减少标签使用计数（补偿操作）
//
// DTM SAGA 分支的补偿操作，用于减少标签的使用计数。
// 使用子事务屏障（barrier）解决：幂等性、空补偿、悬挂问题。
//
// 注意：此方法仅供 DTM Server 调用
func (l *DecrTagUsageCountLogic) DecrTagUsageCount(in *pb.TagUsageCountReq) (*pb.TagUsageCountResp, error) {
	l.Infof("[DTM-Branch] DecrTagUsageCount 开始: tag_ids=%v, delta=%d", in.TagIds, in.Delta)

	if len(in.TagIds) == 0 {
		return &pb.TagUsageCountResp{Success: true}, nil
	}

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
		// 批量减少标签使用计数
		for _, tagID := range in.TagIds {
			_, err := tx.ExecContext(l.ctx, `
				UPDATE interest_tags
				SET usage_count = GREATEST(usage_count - ?, 0)
				WHERE tag_id = ?
			`, in.Delta, tagID)
			if err != nil {
				l.Errorf("[DTM-Branch] 补偿更新标签计数失败: tag_id=%d, err=%v", tagID, err)
				return err
			}
		}
		l.Infof("[DTM-Branch] 标签计数补偿成功: tag_ids=%v, delta=%d", in.TagIds, in.Delta)
		return nil
	})

	if err != nil {
		l.Errorf("[DTM-Branch] DecrTagUsageCount 失败: %v", err)
		return nil, status.Error(codes.Aborted, err.Error())
	}

	return &pb.TagUsageCountResp{Success: true}, nil
}
