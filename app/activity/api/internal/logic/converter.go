// Package logic 提供 HTTP 类型与 RPC 类型之间的转换函数
package logic

import (
	"activity-platform/app/activity/api/internal/types"
	"activity-platform/app/activity/rpc/activityservice"
)

// ==================== Tag 转换 ====================

// ConvertRpcTagToApiTag 将 RPC Tag 转换为 API Tag
func ConvertRpcTagToApiTag(rpcTag *activityservice.Tag) types.Tag {
	if rpcTag == nil {
		return types.Tag{}
	}
	return types.Tag{
		Id:    rpcTag.Id,
		Name:  rpcTag.Name,
		Color: rpcTag.Color,
		Icon:  rpcTag.Icon,
	}
}

// ConvertRpcTagsToApiTags 批量转换 RPC Tags 到 API Tags
func ConvertRpcTagsToApiTags(rpcTags []*activityservice.Tag) []types.Tag {
	if len(rpcTags) == 0 {
		return []types.Tag{}
	}
	result := make([]types.Tag, 0, len(rpcTags))
	for _, tag := range rpcTags {
		result = append(result, ConvertRpcTagToApiTag(tag))
	}
	return result
}

// ==================== Category 转换 ====================

// ConvertRpcCategoryToApiCategory 将 RPC Category 转换为 API Category
func ConvertRpcCategoryToApiCategory(rpcCat *activityservice.Category) types.Category {
	if rpcCat == nil {
		return types.Category{}
	}
	return types.Category{
		Id:   rpcCat.Id,
		Name: rpcCat.Name,
		Icon: rpcCat.Icon,
		Sort: rpcCat.Sort,
	}
}

// ConvertRpcCategoriesToApiCategories 批量转换
func ConvertRpcCategoriesToApiCategories(rpcCats []*activityservice.Category) []types.Category {
	if len(rpcCats) == 0 {
		return []types.Category{}
	}
	result := make([]types.Category, 0, len(rpcCats))
	for _, cat := range rpcCats {
		result = append(result, ConvertRpcCategoryToApiCategory(cat))
	}
	return result
}

// ==================== ActivityDetail 转换 ====================

// ConvertRpcActivityDetailToApi 将 RPC ActivityDetail 转换为 API ActivityDetail
func ConvertRpcActivityDetailToApi(rpc *activityservice.ActivityDetail) types.ActivityDetail {
	if rpc == nil {
		return types.ActivityDetail{}
	}
	return types.ActivityDetail{
		Id:                   rpc.Id,
		Title:                rpc.Title,
		CoverUrl:             rpc.CoverUrl,
		CoverType:            rpc.CoverType,
		Content:              rpc.Content,
		CategoryId:           rpc.CategoryId,
		CategoryName:         rpc.CategoryName,
		OrganizerId:          rpc.OrganizerId,
		OrganizerName:        rpc.OrganizerName,
		OrganizerAvatar:      rpc.OrganizerAvatar,
		ContactPhone:         rpc.ContactPhone,
		RegisterStartTime:    rpc.RegisterStartTime,
		RegisterEndTime:      rpc.RegisterEndTime,
		ActivityStartTime:    rpc.ActivityStartTime,
		ActivityEndTime:      rpc.ActivityEndTime,
		Location:             rpc.Location,
		AddressDetail:        rpc.AddressDetail,
		Longitude:            rpc.Longitude,
		Latitude:             rpc.Latitude,
		MaxParticipants:      rpc.MaxParticipants,
		CurrentParticipants:  rpc.CurrentParticipants,
		RequireApproval:      rpc.RequireApproval,
		RequireStudentVerify: rpc.RequireStudentVerify,
		MinCreditScore:       rpc.MinCreditScore,
		Status:               rpc.Status,
		StatusText:           rpc.StatusText,
		RejectReason:         rpc.RejectReason,
		ViewCount:            rpc.ViewCount,
		LikeCount:            rpc.LikeCount,
		Tags:                 ConvertRpcTagsToApiTags(rpc.Tags),
		CreatedAt:            rpc.CreatedAt,
		UpdatedAt:            rpc.UpdatedAt,
		Version:              rpc.Version,
	}
}

// ==================== ActivityListItem 转换 ====================

// ConvertRpcActivityListItemToApi 将 RPC ActivityListItem 转换为 API ActivityListItem
func ConvertRpcActivityListItemToApi(rpc *activityservice.ActivityListItem) types.ActivityListItem {
	if rpc == nil {
		return types.ActivityListItem{}
	}
	return types.ActivityListItem{
		Id:                  rpc.Id,
		Title:               rpc.Title,
		CoverUrl:            rpc.CoverUrl,
		CoverType:           rpc.CoverType,
		CategoryName:        rpc.CategoryName,
		OrganizerName:       rpc.OrganizerName,
		OrganizerAvatar:     rpc.OrganizerAvatar,
		ActivityStartTime:   rpc.ActivityStartTime,
		Location:            rpc.Location,
		MaxParticipants:     rpc.MaxParticipants,
		CurrentParticipants: rpc.CurrentParticipants,
		Status:              rpc.Status,
		StatusText:          rpc.StatusText,
		Tags:                ConvertRpcTagsToApiTags(rpc.Tags),
		ViewCount:           rpc.ViewCount,
		CreatedAt:           rpc.CreatedAt,
	}
}

// ConvertRpcActivityListItemsToApi 批量转换
func ConvertRpcActivityListItemsToApi(rpcItems []*activityservice.ActivityListItem) []types.ActivityListItem {
	if len(rpcItems) == 0 {
		return []types.ActivityListItem{}
	}
	result := make([]types.ActivityListItem, 0, len(rpcItems))
	for _, item := range rpcItems {
		result = append(result, ConvertRpcActivityListItemToApi(item))
	}
	return result
}

// ==================== Pagination 转换 ====================

// ConvertRpcPaginationToApi 将 RPC Pagination 转换为 API Pagination
func ConvertRpcPaginationToApi(rpc *activityservice.Pagination) types.Pagination {
	if rpc == nil {
		return types.Pagination{}
	}
	return types.Pagination{
		Page:       rpc.Page,
		PageSize:   rpc.PageSize,
		Total:      rpc.Total,
		TotalPages: rpc.TotalPages,
	}
}
