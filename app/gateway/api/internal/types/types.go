package types

// ==================== 用户相关类型 ====================

// RegisterRequest 注册请求
type RegisterRequest struct {
	Phone    string `json:"phone" validate:"required,len=11"`
	Password string `json:"password" validate:"required,min=6,max=20"`
	SMSCode  string `json:"smsCode" validate:"required,len=6"`
	Nickname string `json:"nickname,omitempty" validate:"max=50"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Phone    string `json:"phone" validate:"required,len=11"`
	Password string `json:"password" validate:"required"`
}

// LoginBySMSRequest 短信登录请求
type LoginBySMSRequest struct {
	Phone   string `json:"phone" validate:"required,len=11"`
	SMSCode string `json:"smsCode" validate:"required,len=6"`
}

// SendSMSCodeRequest 发送验证码请求
type SendSMSCodeRequest struct {
	Phone string `json:"phone" validate:"required,len=11"`
	Type  int    `json:"type" validate:"required,oneof=1 2 3"` // 1:注册 2:登录 3:修改密码
}

// RefreshTokenRequest 刷新 Token 请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

// UpdateUserInfoRequest 更新用户信息请求
type UpdateUserInfoRequest struct {
	Nickname  string `json:"nickname,omitempty" validate:"max=50"`
	Avatar    string `json:"avatar,omitempty" validate:"max=255"`
	StudentID string `json:"studentId,omitempty" validate:"max=20"`
	College   string `json:"college,omitempty" validate:"max=100"`
	Gender    int    `json:"gender,omitempty" validate:"oneof=0 1 2"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=6,max=20"`
}

// TokenResponse Token 响应
type TokenResponse struct {
	UserID               int64  `json:"userId"`
	AccessToken          string `json:"accessToken"`
	AccessTokenExpireAt  int64  `json:"accessTokenExpireAt"`
	RefreshToken         string `json:"refreshToken"`
	RefreshTokenExpireAt int64  `json:"refreshTokenExpireAt"`
}

// UserInfoResponse 用户信息响应
type UserInfoResponse struct {
	UserID    int64  `json:"userId"`
	Phone     string `json:"phone"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	StudentID string `json:"studentId"`
	College   string `json:"college"`
	Gender    int    `json:"gender"`
	Status    int    `json:"status"`
	CreatedAt int64  `json:"createdAt"`
}

// ==================== 活动相关类型 ====================

// CreateActivityRequest 创建活动请求
type CreateActivityRequest struct {
	Title            string   `json:"title" validate:"required,max=100"`
	Description      string   `json:"description" validate:"required,max=5000"`
	Category         string   `json:"category" validate:"required"`
	CoverImage       string   `json:"coverImage,omitempty" validate:"max=255"`
	Location         string   `json:"location" validate:"required,max=200"`
	StartTime        int64    `json:"startTime" validate:"required"`
	EndTime          int64    `json:"endTime" validate:"required,gtfield=StartTime"`
	RegistrationEnd  int64    `json:"registrationEnd" validate:"required"`
	MaxParticipants  int      `json:"maxParticipants" validate:"required,min=1"`
	Tags             []string `json:"tags,omitempty" validate:"max=5"`
}

// UpdateActivityRequest 更新活动请求
type UpdateActivityRequest struct {
	Title           string   `json:"title,omitempty" validate:"max=100"`
	Description     string   `json:"description,omitempty" validate:"max=5000"`
	CoverImage      string   `json:"coverImage,omitempty" validate:"max=255"`
	Location        string   `json:"location,omitempty" validate:"max=200"`
	StartTime       int64    `json:"startTime,omitempty"`
	EndTime         int64    `json:"endTime,omitempty"`
	RegistrationEnd int64    `json:"registrationEnd,omitempty"`
	MaxParticipants int      `json:"maxParticipants,omitempty" validate:"min=1"`
	Tags            []string `json:"tags,omitempty" validate:"max=5"`
}

// ActivityListRequest 活动列表请求
type ActivityListRequest struct {
	Category string `form:"category,omitempty"`
	Status   int    `form:"status,omitempty"`
	Keyword  string `form:"keyword,omitempty"`
	Page     int    `form:"page,default=1" validate:"min=1"`
	PageSize int    `form:"pageSize,default=20" validate:"min=1,max=100"`
}

// ActivityDetailResponse 活动详情响应
type ActivityDetailResponse struct {
	ID               int64    `json:"id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Category         string   `json:"category"`
	CoverImage       string   `json:"coverImage"`
	Location         string   `json:"location"`
	StartTime        int64    `json:"startTime"`
	EndTime          int64    `json:"endTime"`
	RegistrationEnd  int64    `json:"registrationEnd"`
	MaxParticipants  int      `json:"maxParticipants"`
	CurrentCount     int      `json:"currentCount"`
	Status           int      `json:"status"`
	Tags             []string `json:"tags"`
	OrganizerID      int64    `json:"organizerId"`
	OrganizerName    string   `json:"organizerName"`
	CreatedAt        int64    `json:"createdAt"`
	IsRegistered     bool     `json:"isRegistered"`     // 当前用户是否已报名
	RegistrationID   int64    `json:"registrationId"`   // 报名ID
}

// RegisterActivityRequest 报名活动请求
type RegisterActivityRequest struct {
	ActivityID int64 `json:"activityId" validate:"required"`
}

// VerifyTicketRequest 核销票据请求
type VerifyTicketRequest struct {
	TicketCode string `json:"ticketCode" validate:"required"`
}

// TicketResponse 票据响应
type TicketResponse struct {
	TicketCode   string `json:"ticketCode"`
	ActivityID   int64  `json:"activityId"`
	ActivityName string `json:"activityName"`
	UserID       int64  `json:"userId"`
	UserName     string `json:"userName"`
	Status       int    `json:"status"`
	CreatedAt    int64  `json:"createdAt"`
	UsedAt       int64  `json:"usedAt,omitempty"`
}

// ==================== 聊天相关类型 ====================

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	RoomID  int64  `json:"roomId" validate:"required"`
	Content string `json:"content" validate:"required,max=500"`
	Type    int    `json:"type,default=1"` // 1:文本 2:图片
}

// MessageResponse 消息响应
type MessageResponse struct {
	ID        int64  `json:"id"`
	RoomID    int64  `json:"roomId"`
	UserID    int64  `json:"userId"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Content   string `json:"content"`
	Type      int    `json:"type"`
	CreatedAt int64  `json:"createdAt"`
}

// RoomMemberResponse 聊天室成员响应
type RoomMemberResponse struct {
	UserID   int64  `json:"userId"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Role     int    `json:"role"` // 1:普通成员 2:管理员 3:创建者
	JoinedAt int64  `json:"joinedAt"`
}

// ==================== 通用类型 ====================

// IDRequest ID 请求（路径参数）
type IDRequest struct {
	ID int64 `path:"id" validate:"required,min=1"`
}

// PageRequest 分页请求
type PageRequest struct {
	Page     int `form:"page,default=1" validate:"min=1"`
	PageSize int `form:"pageSize,default=20" validate:"min=1,max=100"`
}

// EmptyResponse 空响应
type EmptyResponse struct{}

// SuccessResponse 成功响应
type SuccessResponse struct {
	Success bool `json:"success"`
}
