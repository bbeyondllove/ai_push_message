package utils

import (
	"encoding/json"
	"net/http"

	"ai_push_message/models"
)

// WriteFormattedJSON 格式化JSON输出，使其更易读
func WriteFormattedJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "    ") // 使用4个空格缩进
	encoder.Encode(data)
}

// WriteSuccessResponse 写入成功响应
func WriteSuccessResponse(w http.ResponseWriter, data interface{}) {
	WriteFormattedJSON(w, models.NewSuccessResponse(data))
}

// WriteErrorResponse 写入错误响应
func WriteErrorResponse(w http.ResponseWriter, code int, data interface{}) {
	WriteFormattedJSON(w, models.NewErrorResponse(code, data))
}

// WriteCustomErrorResponse 写入自定义错误消息的响应
func WriteCustomErrorResponse(w http.ResponseWriter, code int, message string, data interface{}) {
	WriteFormattedJSON(w, models.NewCustomErrorResponse(code, message, data))
}

// HandleServiceError 处理服务层错误的通用函数
func HandleServiceError(w http.ResponseWriter, err error, noDataCode int) {
	if IsSQLNoRowsError(err) {
		WriteErrorResponse(w, noDataCode, map[string]interface{}{})
	} else {
		WriteCustomErrorResponse(w, models.CodeServerError, err.Error(), map[string]interface{}{})
	}
}

// ValidateCID 验证CID参数
func ValidateCID(w http.ResponseWriter, cid string) bool {
	if cid == "" {
		WriteErrorResponse(w, models.CodeMissingParams, map[string]interface{}{
			"param": "cid",
		})
		return false
	}
	return true
}

// FormatRecommendations 将推荐内容转换为响应格式
func FormatRecommendations(recommendations []models.RecommendationItem) []map[string]string {
	tags := make([]map[string]string, 0, len(recommendations))
	for _, item := range recommendations {
		tags = append(tags, map[string]string{
			"title":   item.Title,
			"content": item.Content,
		})
	}
	return tags
}

// ParseProfileData 解析用户画像数据
func ParseProfileData(w http.ResponseWriter, profileRaw string) (map[string]interface{}, bool) {
	var profileData map[string]interface{}
	if err := json.Unmarshal([]byte(profileRaw), &profileData); err != nil {
		WriteCustomErrorResponse(w, models.CodeServerError, "解析画像数据失败: "+err.Error(), map[string]interface{}{})
		return nil, false
	}
	return profileData, true
}
