package captchaservicelogic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"
	"activity-platform/common/utils/captcha"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckCaptchaLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCheckCaptchaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckCaptchaLogic {
	return &CheckCaptchaLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// geetestResponse 极验响应结构
type geetestResponse struct {
	Result      string       `json:"result"`
	Reason      string       `json:"reason"`
	CaptchaArgs *captchaArgs `json:"captcha_args"`
}

type captchaArgs struct {
	CaptchaId     string `json:"captcha_id"`
	LotNumber     string `json:"lot_number"`
	CaptchaOutput string `json:"captcha_output"`
	PassToken     string `json:"pass_token"`
	GenTime       string `json:"gen_time"`
}

func (l *CheckCaptchaLogic) CheckCaptcha(in *pb.CheckCaptchaReq) (*pb.CheckCaptchaResponse, error) {
	// 1. 获取验证码配置
	captchaId := l.svcCtx.Config.Captcha.CaptchaId
	captchaKey := l.svcCtx.Config.Captcha.CaptchaKey

	if captchaId == "" || captchaKey == "" {
		l.Errorf("验证码配置缺失")
		return nil, errorx.New(errorx.CodeGeetestConfigError)
	}

	// 2. 生成签名
	// sign_token = hmac_sha256(lot_number, captcha_key)
	signToken := captcha.HmacSha256(in.LotNumber, captchaKey)

	// 3. 准备表单数据
	formData := url.Values{}
	formData.Set("lot_number", in.LotNumber)
	formData.Set("captcha_output", in.CaptchaOutput)
	formData.Set("pass_token", in.PassToken)
	formData.Set("gen_time", in.GenTime)
	formData.Set("sign_token", signToken)

	// 4. 发送请求到极验
	// 接口地址：http://gcaptcha4.geetest.com/validate
	apiURL := fmt.Sprintf("http://gcaptcha4.geetest.com/validate?captcha_id=%s", captchaId)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	l.Infof("开始请求极验二次校验: url=%s, params=%v", apiURL, formData)

	resp, err := client.PostForm(apiURL, formData)
	if err != nil {
		l.Errorf("请求极验接口失败: %v", err)
		return nil, errorx.ErrInternalError()
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		l.Errorf("读取极验响应失败: %v", err)
		return nil, errorx.ErrInternalError()
	}

	l.Infof("极验响应: %s", string(body))

	var gtResp geetestResponse
	if err := json.Unmarshal(body, &gtResp); err != nil {
		l.Errorf("解析极验响应失败: %v, body: %s", err, string(body))
		return nil, errorx.ErrInternalError()
	}

	// 5. 构造返回结果
	rpcResp := &pb.CheckCaptchaResponse{
		Result: gtResp.Result,
		Reason: gtResp.Reason,
	}

	// 填充 CaptchaArgs（如果有）
	if gtResp.CaptchaArgs != nil {
		rpcResp.CaptchaArgs = &pb.CaptchaArgs{
			CaptchaId:     gtResp.CaptchaArgs.CaptchaId,
			LotNumber:     gtResp.CaptchaArgs.LotNumber,
			CaptchaOutput: gtResp.CaptchaArgs.CaptchaOutput,
			PassToken:     gtResp.CaptchaArgs.PassToken,
			GenTime:       gtResp.CaptchaArgs.GenTime,
		}
	}

	// 如果校验失败，记录日志
	if gtResp.Result != "success" {
		l.Errorf("极验二次校验失败: result=%s, reason=%s", gtResp.Result, gtResp.Reason)
	}

	return rpcResp, nil
}
