package constants

// API error codes.
const (
	CodeOK                 = 0
	CodeBadRequest         = 40000
	CodeUnauthorized       = 40100
	CodeInvalidCredentials = 40101
	CodeForbidden          = 40300
	CodeNotFound           = 40400
	CodeConflict           = 40900
	CodeStateConflict      = 40901
	CodeTooManyRequests    = 42900
	CodeValidationFailed   = 42200
	CodeInternalError      = 50000
	CodeUserExists         = 40001
)

const (
	MsgOK                 = "ok"
	MsgUnauthorized       = "未登录或登录已过期"
	MsgForbidden          = "没有权限执行该操作"
	MsgNotFound           = "资源不存在"
	MsgTooManyRequests    = "请求过于频繁，请稍后再试"
	MsgInternalError      = "服务器内部错误"
	MsgPhoneExists        = "手机号已注册"
	MsgInvalidCredentials = "手机号或密码错误"
	MsgLoginSuccess       = "登录成功"
	MsgDecisionRecorded   = "放行决定已记录"
)

const (
	RoleAdmin         = "admin"
	RoleSafetyManager = "safety_manager"
	RoleInspector     = "inspector"
	RoleWorker        = "worker"
)

var UserRoleValues = []string{RoleAdmin, RoleSafetyManager, RoleInspector, RoleWorker}

// UnitState is shared with frontend/src/constants/enums.ts.
const (
	UnitAvailable  = "available"
	UnitInspection = "inspection"
	UnitBlocked    = "blocked"
	UnitRetired    = "retired"
)

var UnitStateValues = []string{UnitAvailable, UnitInspection, UnitBlocked, UnitRetired}

// ClearanceState is shared with frontend/src/constants/enums.ts.
const (
	ClearancePending    = "pending"
	ClearanceCleared    = "cleared"
	ClearanceRestricted = "restricted"
	ClearanceRevoked    = "revoked"
)

var ClearanceStateValues = []string{ClearancePending, ClearanceCleared, ClearanceRestricted, ClearanceRevoked}

const (
	TurnaroundOpen       = "open"
	TurnaroundChecking   = "checking"
	TurnaroundDecisioned = "decisioned"
	TurnaroundCompleted  = "completed"
)

var TurnaroundStatusValues = []string{TurnaroundOpen, TurnaroundChecking, TurnaroundDecisioned, TurnaroundCompleted}

const (
	CheckPending = "pending"
	CheckPassed  = "passed"
	CheckFailed  = "failed"
)

var CheckResultValues = []string{CheckPending, CheckPassed, CheckFailed}

const (
	RiskLow      = "low"
	RiskMedium   = "medium"
	RiskHigh     = "high"
	RiskCritical = "critical"
)

var RiskLevelValues = []string{RiskLow, RiskMedium, RiskHigh, RiskCritical}

const (
	LogAuditWriteFailed       = "audit log write failed"
	LogUserLoginSuccess       = "user login success"
	LogUserLoginFailed        = "user login failed"
	LogUserRegisterSuccess    = "user register success"
	LogUserProfileUpdate      = "user profile update"
	LogTurnaroundCreated      = "turnaround created"
	LogTurnaroundUpdated      = "turnaround updated"
	LogGroundUnitCreated      = "ground unit created"
	LogGroundUnitStateChanged = "ground unit state changed"
	LogSafetyCheckCreated     = "safety check created"
	LogSafetyCheckReviewed    = "safety check reviewed"
	LogClearanceChanged       = "clearance state changed"
)

func In(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func IsValidClearanceState(value string) bool { return In(ClearanceStateValues, value) }
func IsValidUnitState(value string) bool      { return In(UnitStateValues, value) }
func IsValidRiskLevel(value string) bool      { return In(RiskLevelValues, value) }
