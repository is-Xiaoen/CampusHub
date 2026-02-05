package activitybranchservicelogic

import (
	"context"
	"database/sql"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/dtm-labs/client/dtmgrpc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CreateActivityActionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateActivityActionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateActivityActionLogic {
	return &CreateActivityActionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CreateActivityAction 创建活动 - 正向操作
//
// DTM SAGA 分支的正向操作，负责在数据库中创建活动记录。
// 使用子事务屏障（barrier）解决以下问题：
//   - 幂等性：重复请求不会重复创建活动
//   - 悬挂：如果补偿操作先到达，正向操作会被跳过
//
// 注意：此方法仅供 DTM Server 调用，不对外暴露
func (l *CreateActivityActionLogic) CreateActivityAction(in *activity.CreateActivityActionReq) (*activity.CreateActivityActionResp, error) {
	l.Infof("[DTM-Branch] CreateActivityAction 开始: organizer_id=%d, title=%s",
		in.OrganizerId, in.Title)

	// 1. 从 gRPC context 获取 DTM 事务信息
	barrier, err := dtmgrpc.BarrierFromGrpc(l.ctx)
	if err != nil {
		l.Errorf("[DTM-Branch] 获取 barrier 失败: %v", err)
		return nil, status.Error(codes.Internal, "获取事务屏障失败")
	}

	// 2. 获取原生 SQL DB（barrier 需要 *sql.DB）
	sqlDB, err := l.svcCtx.DB.DB()
	if err != nil {
		l.Errorf("[DTM-Branch] 获取 SQL DB 失败: %v", err)
		return nil, status.Error(codes.Internal, "获取数据库连接失败")
	}

	var activityID int64
	var activityStatus int32

	// 3. 使用 barrier.CallWithDB 执行业务逻辑
	// barrier 会自动处理幂等和悬挂问题
	err = barrier.CallWithDB(sqlDB, func(tx *sql.Tx) error {
		// 3.1 确定活动状态
		actStatus := int8(in.Status)
		if actStatus == 0 {
			// 未指定状态，根据 IsDraft 判断
			if in.IsDraft {
				actStatus = model.StatusDraft
			} else {
				actStatus = model.StatusPublished
			}
		}

		// 3.2 构建活动实体
		now := time.Now().Unix()
		act := &model.Activity{
			Title:                in.Title,
			CoverURL:             in.CoverUrl,
			CoverType:            int8(in.CoverType),
			Description:          in.Content,
			CategoryID:           uint64(in.CategoryId),
			OrganizerID:          uint64(in.OrganizerId),
			OrganizerName:        in.OrganizerName,
			OrganizerAvatar:      in.OrganizerAvatar,
			ContactPhone:         in.ContactPhone,
			RegisterStartTime:    in.RegisterStartTime,
			RegisterEndTime:      in.RegisterEndTime,
			ActivityStartTime:    in.ActivityStartTime,
			ActivityEndTime:      in.ActivityEndTime,
			Location:             in.Location,
			AddressDetail:        in.AddressDetail,
			Longitude:            in.Longitude,
			Latitude:             in.Latitude,
			MaxParticipants:      uint32(in.MaxParticipants),
			CurrentParticipants:  0,
			RequireApproval:      in.RequireApproval,
			RequireStudentVerify: in.RequireStudentVerify,
			MinCreditScore:       int(in.MinCreditScore),
			Status:               actStatus,
			Version:              0,
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		// 3.2 执行 SQL 插入（使用原生 SQL）
		result, err := tx.ExecContext(l.ctx, `
			INSERT INTO activities (
				title, cover_url, cover_type, description, category_id,
				organizer_id, organizer_name, organizer_avatar, contact_phone,
				register_start_time, register_end_time, activity_start_time, activity_end_time,
				location, address_detail, longitude, latitude,
				max_participants, current_participants, require_approval, require_student_verify, min_credit_score,
				status, version, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			act.Title, act.CoverURL, act.CoverType, act.Description, act.CategoryID,
			act.OrganizerID, act.OrganizerName, act.OrganizerAvatar, act.ContactPhone,
			act.RegisterStartTime, act.RegisterEndTime, act.ActivityStartTime, act.ActivityEndTime,
			act.Location, act.AddressDetail, act.Longitude, act.Latitude,
			act.MaxParticipants, act.CurrentParticipants, act.RequireApproval, act.RequireStudentVerify, act.MinCreditScore,
			act.Status, act.Version, act.CreatedAt, act.UpdatedAt,
		)
		if err != nil {
			l.Errorf("[DTM-Branch] 插入活动失败: %v", err)
			return err
		}

		// 3.4 获取插入的活动 ID
		id, err := result.LastInsertId()
		if err != nil {
			l.Errorf("[DTM-Branch] 获取活动 ID 失败: %v", err)
			return err
		}

		activityID = id
		activityStatus = int32(act.Status)

		// 3.5 绑定标签（如果有）
		if len(in.TagIds) > 0 {
			for _, tagID := range in.TagIds {
				_, err := tx.ExecContext(l.ctx, `
					INSERT INTO activity_tags (activity_id, tag_id, created_at)
					VALUES (?, ?, ?)
				`, activityID, tagID, now)
				if err != nil {
					l.Errorf("[DTM-Branch] 绑定标签失败: activity_id=%d, tag_id=%d, err=%v",
						activityID, tagID, err)
					return err
				}
			}
			l.Infof("[DTM-Branch] 标签绑定成功: activity_id=%d, tags=%v", activityID, in.TagIds)
		}

		l.Infof("[DTM-Branch] 活动创建成功: activity_id=%d, status=%d", activityID, activityStatus)
		return nil
	})

	// 4. 处理 barrier 执行结果
	if err != nil {
		// barrier 内部错误，需要返回 Aborted 触发回滚
		l.Errorf("[DTM-Branch] CreateActivityAction 失败: %v", err)
		return nil, status.Error(codes.Aborted, err.Error())
	}

	// 5. 返回成功响应
	return &activity.CreateActivityActionResp{
		ActivityId: activityID,
		Status:     activityStatus,
	}, nil
}
