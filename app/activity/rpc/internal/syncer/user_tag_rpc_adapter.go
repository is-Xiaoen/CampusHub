package syncer

import (
	"context"

	"activity-platform/app/user/rpc/client/tagservice"
)

// ==================== 用户服务 RPC 适配器 ====================
//
// 用途：将用户服务的 TagService RPC 客户端适配到 UserTagRPCClient 接口
//
// 设计原因：
//   - 解耦：TagReconciler 不直接依赖 proto 生成的代码
//   - 可测试：可以用 mock 实现替换
//   - 可扩展：可以添加缓存、重试、熔断等逻辑

// UserTagRPCAdapter 用户服务标签 RPC 适配器
type UserTagRPCAdapter struct {
	tagRpc tagservice.TagService
}

// NewUserTagRPCAdapter 创建适配器
func NewUserTagRPCAdapter(tagRpc tagservice.TagService) *UserTagRPCAdapter {
	return &UserTagRPCAdapter{
		tagRpc: tagRpc,
	}
}

// GetAllTags 获取所有启用的标签
//
// 调用用户服务 GetAllTags RPC，转换为 TagSyncData 列表
func (a *UserTagRPCAdapter) GetAllTags(ctx context.Context) ([]TagSyncData, error) {
	resp, err := a.tagRpc.GetAllTags(ctx, &tagservice.GetAllTagsReq{
		SinceTimestamp: 0, // 全量获取
	})
	if err != nil {
		return nil, err
	}

	result := make([]TagSyncData, len(resp.Tags))
	for i, tag := range resp.Tags {
		result[i] = TagSyncData{
			ID:          tag.Id,
			Name:        tag.Name,
			Color:       tag.Color,
			Icon:        tag.Icon,
			Status:      int8(tag.Status),
			Description: tag.Description,
			UpdatedAt:   tag.UpdatedAt,
		}
	}

	return result, nil
}

// GetTagsByIDs 根据 ID 批量获取标签
//
// 调用用户服务 GetTagsByIDs RPC，转换为 TagSyncData 列表
func (a *UserTagRPCAdapter) GetTagsByIDs(ctx context.Context, ids []uint64) ([]TagSyncData, error) {
	if len(ids) == 0 {
		return []TagSyncData{}, nil
	}

	// 转换为 int64 切片（proto 通常使用 int64）
	int64IDs := make([]int64, len(ids))
	for i, id := range ids {
		int64IDs[i] = int64(id)
	}

	resp, err := a.tagRpc.GetTagsByIds(ctx, &tagservice.GetTagsByIdsReq{
		Ids: int64IDs,
	})
	if err != nil {
		return nil, err
	}

	result := make([]TagSyncData, len(resp.Tags))
	for i, tag := range resp.Tags {
		result[i] = TagSyncData{
			ID:          tag.Id,
			Name:        tag.Name,
			Color:       tag.Color,
			Icon:        tag.Icon,
			Status:      int8(tag.Status),
			Description: tag.Description,
			UpdatedAt:   tag.UpdatedAt,
		}
	}

	return result, nil
}

// ==================== Mock 实现（用于测试） ====================

// MockUserTagRPCClient Mock 实现
//
// 用于单元测试，可以预设返回数据
type MockUserTagRPCClient struct {
	Tags []TagSyncData
	Err  error
}

// NewMockUserTagRPCClient 创建 Mock 客户端
func NewMockUserTagRPCClient(tags []TagSyncData) *MockUserTagRPCClient {
	return &MockUserTagRPCClient{
		Tags: tags,
	}
}

// GetAllTags 返回预设的标签列表
func (m *MockUserTagRPCClient) GetAllTags(ctx context.Context) ([]TagSyncData, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Tags, nil
}

// GetTagsByIDs 根据 ID 过滤预设的标签列表
func (m *MockUserTagRPCClient) GetTagsByIDs(ctx context.Context, ids []uint64) ([]TagSyncData, error) {
	if m.Err != nil {
		return nil, m.Err
	}

	idSet := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}

	result := make([]TagSyncData, 0, len(ids))
	for _, tag := range m.Tags {
		if _, ok := idSet[tag.ID]; ok {
			result = append(result, tag)
		}
	}

	return result, nil
}
