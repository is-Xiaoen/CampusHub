/**
 * @projectName: CampusHub
 * @package: constants
 * @className: credit
 * @author: lijunqi
 * @description: 信用分相关常量定义
 * @date: 2026-01-30
 * @version: 1.0
 */

package constants

// ==================== 信用等级常量 ====================
// 等级划分说明：
//   Level 0: 黑名单      (score < 60)     - 禁止报名、禁止发布
//   Level 1: 信用风险    (60 <= score < 70) - 每日限报名1次、禁止发布
//   Level 2: 良好用户    (70 <= score < 90) - 正常报名、禁止发布
//   Level 3: 优秀用户    (90 <= score < 95) - 正常报名、允许发布
//   Level 4: 社区之星    (score >= 95)     - 正常报名、允许发布、优先展示
//   新用户初始100分，属于 Level 4

const (
	// CreditLevelBlacklist 黑名单 (score < 60)
	CreditLevelBlacklist int8 = 0
	// CreditLevelRisk 信用风险 (60 <= score < 70)
	CreditLevelRisk int8 = 1
	// CreditLevelGood 良好用户 (70 <= score < 90)
	CreditLevelGood int8 = 2
	// CreditLevelExcellent 优秀用户 (90 <= score < 95)
	CreditLevelExcellent int8 = 3
	// CreditLevelStar 社区之星 (score >= 95)
	CreditLevelStar int8 = 4
)

// CreditLevelNames 信用等级名称映射
var CreditLevelNames = map[int8]string{
	CreditLevelBlacklist: "黑名单",
	CreditLevelRisk:      "信用风险",
	CreditLevelGood:      "良好用户",
	CreditLevelExcellent: "优秀用户",
	CreditLevelStar:      "社区之星",
}

// GetCreditLevelName 获取信用等级名称
func GetCreditLevelName(level int8) string {
	if name, ok := CreditLevelNames[level]; ok {
		return name
	}
	return "未知等级"
}

// ==================== 信用分数阈值 ====================

const (
	// CreditScoreMin 信用分最小值
	CreditScoreMin = 0
	// CreditScoreMax 信用分最大值
	CreditScoreMax = 100
	// CreditScoreInit 初始信用分
	CreditScoreInit = 100

	// CreditThresholdBlacklist 黑名单阈值（低于此分数为黑名单）
	CreditThresholdBlacklist = 60
	// CreditThresholdRisk 风险用户阈值（低于此分数为风险用户）
	CreditThresholdRisk = 70
	// CreditThresholdGood 良好用户阈值（低于此分数为良好用户）
	CreditThresholdGood = 90
	// CreditThresholdExcellent 优秀用户阈值（低于此分数为优秀用户）
	CreditThresholdExcellent = 95
	// CreditThresholdPublish 发布活动所需最低分数
	CreditThresholdPublish = 90
)

// CalculateCreditLevel 根据分数计算信用等级
func CalculateCreditLevel(score int) int8 {
	switch {
	case score < CreditThresholdBlacklist:
		return CreditLevelBlacklist
	case score < CreditThresholdRisk:
		return CreditLevelRisk
	case score < CreditThresholdGood:
		return CreditLevelGood
	case score < CreditThresholdExcellent:
		return CreditLevelExcellent
	default:
		return CreditLevelStar
	}
}

// ==================== 信用变更类型 ====================
// 对应 proto 中的 change_type 字段

const (
	// CreditChangeTypeInit 注册初始化 -> 设为100分
	CreditChangeTypeInit int32 = 1
	// CreditChangeTypeCheckin 正常履约/签到成功 -> +2
	CreditChangeTypeCheckin int32 = 2
	// CreditChangeTypeCancelEarly 提前24h取消 -> 0（无责取消）
	CreditChangeTypeCancelEarly int32 = 3
	// CreditChangeTypeCancelLate 临期取消(<24h) -> -5
	CreditChangeTypeCancelLate int32 = 4
	// CreditChangeTypeNoShow 爽约/未签到 -> -10
	CreditChangeTypeNoShow int32 = 5
	// CreditChangeTypeHostSuccess 圆满举办活动 -> +5（组织者奖励）
	CreditChangeTypeHostSuccess int32 = 6
	// CreditChangeTypeHostDelete 删除已有报名的活动 -> -10（组织者惩罚）
	CreditChangeTypeHostDelete int32 = 7
	// CreditChangeTypeAdminAdjust 管理员人工调整 -> 分值由admin_delta指定
	CreditChangeTypeAdminAdjust int32 = 99
)

// CreditChangeDeltas 信用变更类型对应的分值变动
var CreditChangeDeltas = map[int32]int{
	CreditChangeTypeInit:        100,
	CreditChangeTypeCheckin:     2,
	CreditChangeTypeCancelEarly: 0,
	CreditChangeTypeCancelLate:  -5,
	CreditChangeTypeNoShow:      -10,
	CreditChangeTypeHostSuccess: 5,
	CreditChangeTypeHostDelete:  -10,
}

// CreditChangeTypeNames 信用变更类型名称映射
var CreditChangeTypeNames = map[int]string{
	int(CreditChangeTypeInit):        "注册初始化",
	int(CreditChangeTypeCheckin):     "正常履约",
	int(CreditChangeTypeCancelEarly): "提前取消",
	int(CreditChangeTypeCancelLate):  "临期取消",
	int(CreditChangeTypeNoShow):      "爽约",
	int(CreditChangeTypeHostSuccess): "圆满举办",
	int(CreditChangeTypeHostDelete):  "删除活动",
	int(CreditChangeTypeAdminAdjust): "管理员调整",
}

// GetCreditChangeTypeName 获取信用变更类型名称
func GetCreditChangeTypeName(changeType int) string {
	if name, ok := CreditChangeTypeNames[changeType]; ok {
		return name
	}
	return "未知类型"
}

// GetCreditDelta 获取信用变更类型对应的分值变动
// changeType: 变更类型
// adminDelta: 管理员调整时的自定义分值
func GetCreditDelta(changeType int32, adminDelta int64) int {
	if changeType == CreditChangeTypeAdminAdjust {
		return int(adminDelta)
	}
	if delta, ok := CreditChangeDeltas[changeType]; ok {
		return delta
	}
	return 0
}

// ==================== 风险用户限制 ====================

const (
	// RiskUserDailyParticipateLimit 风险用户每日报名次数限制
	RiskUserDailyParticipateLimit = 1
)

// ==================== Redis Key 相关 ====================

const (
	// CacheUserCreditPrefix 用户信用信息缓存前缀
	CacheUserCreditPrefix = "user:credit:"
	// CacheRiskUserDailyCountPrefix 风险用户每日报名计数前缀
	// 格式: risk:participate:daily:{userId}:{date}
	CacheRiskUserDailyCountPrefix = "risk:participate:daily:"
)
