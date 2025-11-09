package services

import (
	"ai_push_message/config"
	"ai_push_message/models"
)

// ProfileService 用户画像服务接口
type ProfileService interface {
	// 为指定用户生成画像
	GenerateProfileForUser(cfg *config.Config, cid string) error

	// 为所有用户生成画像
	GenerateProfileForAllUsers(cfg *config.Config) error
	GenerateProfilesWithConcurrency(cfg *config.Config, cids []string, concurrency int)

	// 验证用户是否有有效画像
	ValidateUserProfile(cid string) (bool, error)
}

// RecommendationService 推荐内容服务接口
type RecommendationService interface {
	// 为指定用户生成推荐内容
	GenerateRecommendationsForUser(cfg *config.Config, cid string) error

	// 为所有用户生成推荐内容
	GenerateRecommendationsForAllUsers(cfg *config.Config) error

	GenerateRecommendationsWithConcurrency(cfg *config.Config, cids []string, concurrency int)

	// 获取用户的推荐内容
	GetUserRecommendations(cid string) ([]models.RecommendationItem, error)

	// 刷新用户推荐内容（实时触发）
	RefreshUserRecommendations(cfg *config.Config, cid string) ([]models.RecommendationItem, error)

	// 获取推荐统计信息
	GetRecommendationStats() map[string]interface{}
}

// RAGService RAG服务接口
type RAGService interface {
	// 调用RAG服务搜索
	CallRAG(cfg *config.Config, query string) ([]models.RecommendationItem, error)
}

// PushService 推送服务接口
type PushService interface {
	// 为指定用户推送内容
	PushForCID(cfg *config.Config, cid string) error

	// 为所有用户推送内容
	PushAll(cfg *config.Config) error
}
