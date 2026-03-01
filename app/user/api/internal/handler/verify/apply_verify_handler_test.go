package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/verifyservice"
	"activity-platform/common/response"

	"google.golang.org/grpc"
)

const testUserID int64 = 10086

type mockVerifyService struct {
	applyCalled int
	applyReq    *verifyservice.ApplyStudentVerifyReq
	applyResp   *verifyservice.ApplyStudentVerifyResp
	applyErr    error
}

func (m *mockVerifyService) GetVerifyCurrent(ctx context.Context, in *verifyservice.GetVerifyCurrentReq, opts ...grpc.CallOption) (*verifyservice.GetVerifyCurrentResp, error) {
	panic("unexpected GetVerifyCurrent call")
}

func (m *mockVerifyService) GetVerifyInfo(ctx context.Context, in *verifyservice.GetVerifyInfoReq, opts ...grpc.CallOption) (*verifyservice.GetVerifyInfoResp, error) {
	panic("unexpected GetVerifyInfo call")
}

func (m *mockVerifyService) IsVerified(ctx context.Context, in *verifyservice.IsVerifiedReq, opts ...grpc.CallOption) (*verifyservice.IsVerifiedResp, error) {
	panic("unexpected IsVerified call")
}

func (m *mockVerifyService) ApplyStudentVerify(ctx context.Context, in *verifyservice.ApplyStudentVerifyReq, opts ...grpc.CallOption) (*verifyservice.ApplyStudentVerifyResp, error) {
	m.applyCalled++
	m.applyReq = in
	return m.applyResp, m.applyErr
}

func (m *mockVerifyService) ConfirmStudentVerify(ctx context.Context, in *verifyservice.ConfirmStudentVerifyReq, opts ...grpc.CallOption) (*verifyservice.ConfirmStudentVerifyResp, error) {
	panic("unexpected ConfirmStudentVerify call")
}

func (m *mockVerifyService) CancelStudentVerify(ctx context.Context, in *verifyservice.CancelStudentVerifyReq, opts ...grpc.CallOption) (*verifyservice.CancelStudentVerifyResp, error) {
	panic("unexpected CancelStudentVerify call")
}

func (m *mockVerifyService) UpdateVerifyStatus(ctx context.Context, in *verifyservice.UpdateVerifyStatusReq, opts ...grpc.CallOption) (*verifyservice.UpdateVerifyStatusResp, error) {
	panic("unexpected UpdateVerifyStatus call")
}

func (m *mockVerifyService) ProcessOcrVerify(ctx context.Context, in *verifyservice.ProcessOcrVerifyReq, opts ...grpc.CallOption) (*verifyservice.ProcessOcrVerifyResp, error) {
	panic("unexpected ProcessOcrVerify call")
}

type apiResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func TestApplyVerifyHandler_MissingField(t *testing.T) {
	response.SetupGlobalErrorHandler()

	mockRpc := &mockVerifyService{}
	handler := ApplyVerifyHandler(&svc.ServiceContext{
		VerifyServiceRpc: mockRpc,
	})

	body := `{
		"real_name":"张三",
		"school_name":"华中科技大学",
		"student_id":"U202312345",
		"department":"计算机科学与技术学院",
		"admission_year":"2023",
		"front_image_url":"https://cdn.example.com/verify/front.jpg"
	}`
	req := newJSONRequest(body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected http status: got=%d want=%d body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	var resp apiResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Code != 1001 {
		t.Fatalf("unexpected biz code: got=%d want=1001", resp.Code)
	}
	if !strings.Contains(resp.Message, "back_image_url") {
		t.Fatalf("unexpected error message: %s", resp.Message)
	}
	if mockRpc.applyCalled != 0 {
		t.Fatalf("apply rpc should not be called, got=%d", mockRpc.applyCalled)
	}
}

func TestApplyVerifyHandler_EmptyStringField(t *testing.T) {
	response.SetupGlobalErrorHandler()

	mockRpc := &mockVerifyService{}
	handler := ApplyVerifyHandler(&svc.ServiceContext{
		VerifyServiceRpc: mockRpc,
	})

	body := `{
		"real_name":"张三",
		"school_name":"华中科技大学",
		"student_id":"U202312345",
		"department":"   ",
		"admission_year":"2023",
		"front_image_url":"https://cdn.example.com/verify/front.jpg",
		"back_image_url":"https://cdn.example.com/verify/back.jpg"
	}`
	req := withUserID(newJSONRequest(body), testUserID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected http status: got=%d want=%d body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	var resp apiResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Code != 1001 {
		t.Fatalf("unexpected biz code: got=%d want=1001", resp.Code)
	}
	if resp.Message != "院系不能为空" {
		t.Fatalf("unexpected error message: got=%s want=%s", resp.Message, "院系不能为空")
	}
	if mockRpc.applyCalled != 0 {
		t.Fatalf("apply rpc should not be called, got=%d", mockRpc.applyCalled)
	}
}

func TestApplyVerifyHandler_Success(t *testing.T) {
	response.SetupGlobalErrorHandler()
	response.SetupGlobalOkHandler()

	mockRpc := &mockVerifyService{
		applyResp: &verifyservice.ApplyStudentVerifyResp{
			VerifyId:   12345,
			Status:     1,
			StatusDesc: "OCR审核中",
			CreatedAt:  1700000000,
		},
	}
	handler := ApplyVerifyHandler(&svc.ServiceContext{
		VerifyServiceRpc: mockRpc,
	})

	body := `{
		"real_name":"张三",
		"school_name":"华中科技大学",
		"student_id":"U202312345",
		"department":"计算机科学与技术学院",
		"admission_year":"2023",
		"front_image_url":"https://cdn.example.com/verify/front.jpg",
		"back_image_url":"https://cdn.example.com/verify/back.jpg"
	}`
	req := withUserID(newJSONRequest(body), testUserID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected http status: got=%d want=%d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp apiResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected biz code: got=%d want=0", resp.Code)
	}
	if resp.Message != "success" {
		t.Fatalf("unexpected message: got=%s want=success", resp.Message)
	}

	var data types.ApplyVerifyResp
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal response data failed: %v", err)
	}
	if data.VerifyId != 12345 {
		t.Fatalf("unexpected verify id: got=%d want=12345", data.VerifyId)
	}
	if data.Status != 1 {
		t.Fatalf("unexpected status: got=%d want=1", data.Status)
	}
	if data.StatusDesc != "OCR审核中" {
		t.Fatalf("unexpected status desc: got=%s want=%s", data.StatusDesc, "OCR审核中")
	}
	expectedTime := time.Unix(1700000000, 0).Format(time.RFC3339)
	if data.CreatedAt != expectedTime {
		t.Fatalf("unexpected created_at: got=%s want=%s", data.CreatedAt, expectedTime)
	}

	if mockRpc.applyCalled != 1 {
		t.Fatalf("apply rpc should be called once, got=%d", mockRpc.applyCalled)
	}
	if mockRpc.applyReq == nil {
		t.Fatal("apply rpc request is nil")
	}
	if mockRpc.applyReq.UserId != testUserID {
		t.Fatalf("unexpected user id: got=%d want=%d", mockRpc.applyReq.UserId, testUserID)
	}
	if mockRpc.applyReq.RealName != "张三" ||
		mockRpc.applyReq.SchoolName != "华中科技大学" ||
		mockRpc.applyReq.StudentId != "U202312345" ||
		mockRpc.applyReq.Department != "计算机科学与技术学院" ||
		mockRpc.applyReq.AdmissionYear != "2023" ||
		mockRpc.applyReq.FrontImageUrl != "https://cdn.example.com/verify/front.jpg" ||
		mockRpc.applyReq.BackImageUrl != "https://cdn.example.com/verify/back.jpg" {
		t.Fatalf("rpc request mapping mismatch: %+v", mockRpc.applyReq)
	}
}

func newJSONRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/verify/student/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func withUserID(req *http.Request, userID int64) *http.Request {
	ctx := context.WithValue(req.Context(), "userId", userID)
	return req.WithContext(ctx)
}
