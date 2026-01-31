/**
 * @projectName: CampusHub
 * @package: creditservicelogic
 * @className: GetCreditLogsLogic
 * @author: lijunqi
 * @description: 获取信用变更记录列表逻辑层
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

	"github.com/zeromicro/go-zero/core/logx"
)

// GetCreditLogsLogic 获取信用变更记录列表逻辑处理器
type GetCreditLogsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewGetCreditLogsLogic 创建获取信用变更记录列表逻辑实例
func NewGetCreditLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCreditLogsLogic {
	return &GetCreditLogsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetCreditLogs 获取信用变更记录列表
// 业务逻辑:
//   - 支持按变动类型、时间范围筛选
//   - 分页查询，按时间倒序排列
func (l *GetCreditLogsLogic) GetCreditLogs(in *pb.GetCreditLogsReq) (*pb.GetCreditLogsResp, error) {
	// 1. 参数校验
	if in.UserId <= 0 {
		l.Errorf("GetCreditLogs 参数错误: userId=%d", in.UserId)
		return nil, errorx.ErrInvalidParams("用户ID无效")
	}

	// 2. 处理分页参数
	page := in.Page
	pageSize := in.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}
	offset := int((page - 1) * pageSize)

	// 3. 构建查询条件
	query := &model.CreditLogQuery{
		UserID:     in.UserId,
		ChangeType: int8(in.ChangeType),
		StartTime:  in.StartTime,
		EndTime:    in.EndTime,
		Offset:     offset,
		Limit:      int(pageSize),
	}

	// 4. 查询总数
	total, err := l.svcCtx.CreditLogModel.CountByQuery(l.ctx, query)
	if err != nil {
		l.Errorf("GetCreditLogs 统计记录数失败: userId=%d, err=%v", in.UserId, err)
		return nil, errorx.ErrDBError(err)
	}

	// 5. 如果没有记录，直接返回空列表
	if total == 0 {
		return &pb.GetCreditLogsResp{
			List:     []*pb.CreditLogItem{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	// 6. 查询记录列表
	logs, err := l.svcCtx.CreditLogModel.ListByQuery(l.ctx, query)
	if err != nil {
		l.Errorf("GetCreditLogs 查询记录列表失败: userId=%d, err=%v", in.UserId, err)
		return nil, errorx.ErrDBError(err)
	}

	// 7. 转换为响应格式
	list := make([]*pb.CreditLogItem, 0, len(logs))
	for _, log := range logs {
		list = append(list, l.convertToProto(log))
	}

	l.Infof("GetCreditLogs 查询成功: userId=%d, total=%d, page=%d, pageSize=%d",
		in.UserId, total, page, pageSize)

	return &pb.GetCreditLogsResp{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// convertToProto 将 Model 转换为 Proto 格式
func (l *GetCreditLogsLogic) convertToProto(log *model.CreditLog) *pb.CreditLogItem {
	return &pb.CreditLogItem{
		Id:             log.ID,
		UserId:         log.UserID,
		ChangeType:     int32(log.ChangeType),
		ChangeTypeName: constants.GetCreditChangeTypeName(int(log.ChangeType)),
		Delta:          int32(log.Delta),
		SourceId:       log.SourceID,
		Reason:         log.Reason,
		BeforeScore:    int32(log.BeforeScore),
		AfterScore:     int32(log.AfterScore),
		CreatedAt:      log.CreatedAt.Unix(),
	}
}
