package models

// WebhookRequest 知识库更新webhook请求体
type WebhookRequest struct {
	KnowledgeID string   `json:"knowledge_id" example:"kb_faq"`
	DocumentIDs []string `json:"document_ids" example:"['doc1', 'doc2']"`
	UpdateType  string   `json:"update_type" example:"add"` // "add", "update", "delete"
	Timestamp   int64    `json:"timestamp" example:"1629123456789"`
}

// APIResponse 通用API响应
type APIResponse struct {
	Code    int         `json:"code" example:"0"`
	Message string      `json:"message" example:"success"`
	Data    interface{} `json:"data,omitempty"`
}

// ProfileResponse 用户画像响应
type ProfileResponse struct {
	Code        int    `json:"code" example:"0"`
	Message     string `json:"message" example:"success"`
	HasProfile  bool   `json:"has_profile" example:"true"`
	ProfileData string `json:"profile_data,omitempty"`
}

// RecommendationResponse 推荐内容响应
type RecommendationResponse struct {
	Code    int                  `json:"code" example:"0"`
	Message string               `json:"message" example:"success"`
	Data    []RecommendationItem `json:"data"`
}