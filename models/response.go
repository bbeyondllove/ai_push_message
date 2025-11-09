package models

// 响应码定义
const (
	// 成功
	CodeSuccess = 0

	// 客户端错误 (1000-1999)
	CodeInvalidParams   = 1000 // 无效的参数
	CodeMissingParams   = 1001 // 缺少必要参数
	CodeUserNotFound    = 1002 // 用户不存在
	CodeNoUserProfile   = 1003 // 用户没有画像
	CodeNoRecommendData = 1004 // 没有推荐数据

	// 服务端错误 (2000-2999)
	CodeServerError        = 2000 // 服务器内部错误
	CodeDatabaseError      = 2001 // 数据库错误
	CodeProfileGenError    = 2002 // 画像生成错误
	CodeRecommendGenError  = 2003 // 推荐生成错误
	CodeThirdPartyAPIError = 2005 // 第三方API错误
)

// 错误码对应的消息
var CodeMessages = map[int]string{
	CodeSuccess:            "success",
	CodeInvalidParams:      "无效的参数",
	CodeMissingParams:      "缺少必要参数",
	CodeUserNotFound:       "用户不存在",
	CodeNoUserProfile:      "用户没有画像",
	CodeNoRecommendData:    "没有推荐数据",
	CodeServerError:        "服务器内部错误",
	CodeDatabaseError:      "数据库错误",
	CodeProfileGenError:    "画像生成错误",
	CodeRecommendGenError:  "推荐生成错误",
	CodeThirdPartyAPIError: "第三方API错误",
}

// 注意：APIResponse结构体已在swagger_models.go中定义，此处不再重复定义

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}) APIResponse {
	return APIResponse{
		Code:    CodeSuccess,
		Message: CodeMessages[CodeSuccess],
		Data:    data,
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code int, data interface{}) APIResponse {
	message, exists := CodeMessages[code]
	if !exists {
		message = "未知错误"
	}
	return APIResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewCustomErrorResponse 创建自定义错误消息的响应
func NewCustomErrorResponse(code int, message string, data interface{}) APIResponse {
	return APIResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
}
