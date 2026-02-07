/**
 * @projectName: CampusHub
 * @package: verify
 * @className: ApplyVerifyLogic
 * @author: lijunqi
 * @description: 提交学生认证申请业务逻辑（先上传图片，再提交认证）
 * @date: 2026-02-07
 * @version: 2.0
 */

package verify

import (
	"context"
	"time"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/uploadtoqiniu"
	"activity-platform/app/user/rpc/client/verifyservice"
	"activity-platform/common/ctxdata"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

// ApplyVerifyLogic 提交学生认证申请逻辑处理器
type ApplyVerifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewApplyVerifyLogic 创建提交学生认证申请逻辑实例
func NewApplyVerifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApplyVerifyLogic {
	return &ApplyVerifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ApplyVerify 提交学生认证申请
// 流程: 1. 上传图片到七牛云 → 2. 提交认证申请
func (l *ApplyVerifyLogic) ApplyVerify(
	req *types.ApplyVerifyReq,
	frontData []byte, frontName string,
	backData []byte, backName string,
) (resp *types.ApplyVerifyResp, err error) {

	// 1. 从 JWT 中获取当前用户ID
	userId := ctxdata.GetUserIDFromCtx(l.ctx)
	if userId <= 0 {
		l.Errorf("ApplyVerify 获取用户ID失败")
		return nil, errorx.ErrUnauthorized()
	}

	// 2.【第一步 RPC】调用 UploadToQiNiuRpc 上传学生证图片
	l.Infof("ApplyVerify 开始上传图片: userId=%d, frontName=%s, backName=%s",
		userId, frontName, backName)

	uploadResp, err := l.svcCtx.UploadToQiNiuRpc.UploadStudentCardImages(l.ctx,
		&uploadtoqiniu.UploadStudentCardImagesReq{
			UserId:         userId,
			FrontImageData: frontData,
			FrontImageName: frontName,
			BackImageData:  backData,
			BackImageName:  backName,
		})
	if err != nil {
		l.Errorf("ApplyVerify 上传图片失败: userId=%d, err=%v", userId, err)
		return nil, errorx.FromError(err)
	}

	l.Infof("ApplyVerify 图片上传成功: userId=%d, frontUrl=%s, backUrl=%s",
		userId, uploadResp.FrontImageUrl, uploadResp.BackImageUrl)

	// 3.【第二步 RPC】调用 VerifyServiceRpc 提交认证申请（使用上传返回的 URL）
	rpcResp, err := l.svcCtx.VerifyServiceRpc.ApplyStudentVerify(l.ctx,
		&verifyservice.ApplyStudentVerifyReq{
			UserId:        userId,
			RealName:      req.RealName,
			SchoolName:    req.SchoolName,
			StudentId:     req.StudentId,
			Department:    req.Department,
			AdmissionYear: req.AdmissionYear,
			FrontImageUrl: uploadResp.FrontImageUrl,
			BackImageUrl:  uploadResp.BackImageUrl,
		})
	if err != nil {
		l.Errorf("ApplyVerify 调用认证 RPC 失败: userId=%d, err=%v", userId, err)
		return nil, errorx.FromError(err)
	}

	// 4. 转换 RPC 响应为 API 响应
	resp = &types.ApplyVerifyResp{
		VerifyId:   rpcResp.VerifyId,
		Status:     rpcResp.Status,
		StatusDesc: rpcResp.StatusDesc,
		CreatedAt:  time.Unix(rpcResp.CreatedAt, 0).Format(time.RFC3339),
	}

	l.Infof("ApplyVerify 提交成功: userId=%d, verifyId=%d, status=%d",
		userId, resp.VerifyId, resp.Status)

	return resp, nil
}
