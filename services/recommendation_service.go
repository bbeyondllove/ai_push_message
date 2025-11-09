package services

import (
	"ai_push_message/config"
	"ai_push_message/logger"
	"ai_push_message/models"
	"ai_push_message/repository"
	"ai_push_message/utils"
	"encoding/json"
	"sync"
	"time"
)

// GenerateRecommendationsForUser 为指定用户生成推荐内容
// 根据用户画像是否重新生成来决定是否需要重新生成推荐
func GenerateRecommendationsForUser(cfg *config.Config, cid string) ([]models.RecommendationItem, error) {
	logger.Info("Checking if recommendations need regeneration for user", "cid", cid)

	// 生成或获取用户画像，同时获得是否重新生成的信息
	profile, profileRegenerated, err := GenerateProfileForUser(cfg, cid)
	if err != nil {
		logger.Error("Failed to generate/get profile", "cid", cid, "error", err)
		return nil, err
	}

	// 如果画像没有重新生成，检查是否已有推荐内容
	if !profileRegenerated {
		existing, err := repository.GetRecommendations(cid)
		if err != nil {
			logger.Info("No existing recommendations found for user", "cid", cid, "error", err.Error())
		} else if len(existing) > 0 {
			logger.Info("Profile not regenerated and user already has recommendations, returning cached", "cid", cid)
			return existing, nil
		}
	}

	logger.Info("Generating new recommendations for user", "cid", cid, "profile_regenerated", profileRegenerated)

	// 直接使用已获取的画像生成推荐，避免重复数据库查询
	recommendations, err := ForceGenerateRecommendationsForUserWithProfile(cfg, cid, profile)
	if err != nil {
		logger.Error("Failed to generate recommendations", "cid", cid, "error", err)
		return nil, err
	}

	logger.Info("Recommendations generated/updated", "cid", cid, "count", len(recommendations))
	return recommendations, nil
}

// GetHotTopicsAsRecommendations 获取群聊热门话题作为推荐内容（只获取前一天的）
func GetHotTopicsAsRecommendations(cfg *config.Config) ([]models.RecommendationItem, error) {
	// 从数据库获取热门话题（只获取前一天的）
	logger.Info("Fetching hot topics from yesterday's group summaries")
	hotTopics, err := repository.GetHotTopicsFromGroupSummaries() // 获取前一天的群聊总结
	if err != nil {
		logger.Error("Failed to get hot topics from group summaries", "error", err)
		return nil, err
	}

	// 创建RAG内容格式化器
	formatter := utils.NewRAGContentFormatter()

	var recommendations []models.RecommendationItem
	for _, topic := range hotTopics {
		// 使用格式化器分别处理标题和内容
		formattedTitle, formattedContent := formatter.FormatTitleAndContent(topic.Title, topic.Content)

		// 转换为推荐内容格式
		item := models.RecommendationItem{
			Source:  "group_summary",
			Title:   formattedTitle,
			Content: formattedContent,
			Score:   1.0, // 给予默认分数
		}
		recommendations = append(recommendations, item)
	}

	// 限制返回数量
	if len(recommendations) > cfg.RAG.TopK {
		recommendations = recommendations[:cfg.RAG.TopK]
	}

	logger.Info("Generated recommendations from yesterday's hot topics", "count", len(recommendations))
	return recommendations, nil
}

// GenerateRecommendationsForAllUsers 并发生成所有用户推荐内容
func GenerateRecommendationsForAllUsers(cfg *config.Config) error {
	logger.Info("Starting recommendation generation for all users")

	cids, err := repository.ListUsersWithProfiles()
	if err != nil {
		return err
	}
	logger.Info("Users with profiles found", "count", len(cids))

	GenerateRecommendationsWithConcurrency(cfg, cids, cfg.Cron.Concurrency)
	return nil
}

// extractKeywords 从用户画像中提取关键词
func extractKeywords(profile *models.UserProfile) []string {
	if profile == nil {
		return []string{}
	}

	var keywords []string

	// 首先尝试从 Keywords 字段解析
	if profile.Keywords != "" {
		if err := json.Unmarshal([]byte(profile.Keywords), &keywords); err == nil && len(keywords) > 0 {
			return keywords
		} else if err != nil {
			logger.Debug("Failed to parse keywords from Keywords field", "cid", profile.CID, "error", err)
		}
	}

	// 如果 Keywords 字段解析失败，尝试从 ProfileRaw 中提取
	if profile.ProfileRaw != "" {
		var profileData map[string]interface{}
		if err := json.Unmarshal([]byte(profile.ProfileRaw), &profileData); err == nil {
			// 从 weighted_keywords 中提取
			if wkRaw, ok := profileData["weighted_keywords"].([]interface{}); ok {
				for _, wk := range wkRaw {
					if kwObj, ok := wk.(map[string]interface{}); ok {
						if kw, ok := kwObj["keyword"].(string); ok && kw != "" {
							keywords = append(keywords, kw)
						}
					}
				}
			}

			// 如果还是没有关键词，尝试从 interests 中获取
			if len(keywords) == 0 {
				if interests, ok := profileData["interests"].([]interface{}); ok {
					for _, interest := range interests {
						if interestStr, ok := interest.(string); ok && interestStr != "" {
							keywords = append(keywords, interestStr)
						}
					}
				}
			}
		} else {
			logger.Debug("Failed to parse ProfileRaw", "cid", profile.CID, "error", err)
		}
	}

	logger.Debug("Extracted keywords from profile", "cid", profile.CID, "count", len(keywords))
	return keywords
}

// SearchKnowledgeBaseByProfile 根据用户画像搜索知识库
func SearchKnowledgeBaseByProfile(cfg *config.Config, keywords []string) ([]models.RecommendationItem, error) {
	allRecommendations := make([]models.RecommendationItem, 0)
	seen := make(map[string]bool)
	processedKeywords := make(map[string]bool)

	logger.Info("Searching knowledge base with keywords", "count", len(keywords))

	// 按权重顺序（从高到低）处理关键词
	for _, keyword := range keywords {
		// 跳过空关键词和已处理的关键词
		if keyword == "" || processedKeywords[keyword] {
			continue
		}

		processedKeywords[keyword] = true
		logger.Info("Searching with keyword", "keyword", keyword)

		// 调用RAG服务搜索
		items, err := CallRAG(cfg, keyword)
		if err != nil {
			logger.Error("RAG search failed for keyword", "keyword", keyword, "error", err)
			continue
		}

		logger.Info("Found items for keyword", "keyword", keyword, "count", len(items))

		// 检查返回内容是否为空
		if len(items) == 0 {
			logger.Info("No items found for keyword, skipping", "keyword", keyword)
			continue
		}

		// 去重并添加到结果中
		for _, item := range items {
			key := item.RefID + "|" + item.Title
			if !seen[key] {
				item.Source = "knowledge_base"
				item.SearchKeyword = keyword
				allRecommendations = append(allRecommendations, item)
				seen[key] = true
			}
		}

		// 如果已经找到足够的推荐内容，就不再继续搜索
		if len(allRecommendations) >= cfg.RAG.TopK {
			logger.Info("Found enough recommendations, stopping search", "count", len(allRecommendations))
			break
		}
	}

	logger.Info("Total recommendations found", "count", len(allRecommendations))

	// 按相关性分数排序并限制数量
	if len(allRecommendations) > cfg.RAG.TopK {
		allRecommendations = allRecommendations[:cfg.RAG.TopK]
	}

	return allRecommendations, nil
}

// GetUserRecommendations 获取用户的推荐内容
func GetUserRecommendations(cid string) ([]models.RecommendationItem, error) {
	return repository.GetRecommendations(cid)
}

// RefreshUserRecommendations 刷新用户推荐内容（实时触发）
// 强制重新生成推荐内容，不检查画像更新时间
func RefreshUserRecommendations(cfg *config.Config, cid string) ([]models.RecommendationItem, error) {
	return RefreshUserRecommendationsWithOptions(cfg, cid, true)
}

// RefreshUserRecommendationsWithOptions 刷新用户推荐内容带选项
// forceProfileRegeneration: 是否强制重新生成画像
func RefreshUserRecommendationsWithOptions(cfg *config.Config, cid string, forceProfileRegeneration bool) ([]models.RecommendationItem, error) {
	logger.Info("Refreshing recommendations for user", "cid", cid, "force_profile_regen", forceProfileRegeneration)

	var profile *models.UserProfile
	var err error

	if forceProfileRegeneration {
		// 强制重新生成用户画像
		profile, _, err = GenerateProfileForUser(cfg, cid)
		if err != nil {
			logger.Error("Failed to refresh profile for user", "cid", cid, "error", err)
			return nil, err
		}
	} else {
		// 先尝试获取现有画像
		profile, err = repository.GetProfile(cid)
		if err != nil {
			// 如果没有画像，再生成
			profile, _, err = GenerateProfileForUser(cfg, cid)
			if err != nil {
				logger.Error("Failed to generate profile for user", "cid", cid, "error", err)
				return nil, err
			}
		}
	}

	// 直接使用获取到的用户画像生成推荐，避免重复数据库查询
	recommendations, err := ForceGenerateRecommendationsForUserWithProfile(cfg, cid, profile)
	if err != nil {
		logger.Error("Failed to refresh recommendations for user", "cid", cid, "error", err)
		return nil, err
	}

	logger.Info("Successfully refreshed recommendations for user", "cid", cid)
	return recommendations, nil
}

// ForceGenerateRecommendationsForUserWithProfile 使用传入的用户画像强制生成推荐内容
// 避免重复的数据库查询，提高性能
func ForceGenerateRecommendationsForUserWithProfile(cfg *config.Config, cid string, profile *models.UserProfile) ([]models.RecommendationItem, error) {
	logger.Info("Force generating recommendations for user with provided profile", "cid", cid)

	var recommendations []models.RecommendationItem
	var err error

	// 如果用户有画像和关键词，优先使用基于画像的推荐
	if profile != nil && profile.Keywords != "" {
		keywords := extractKeywords(profile)
		if len(keywords) > 0 {
			recommendations, err = SearchKnowledgeBaseByProfile(cfg, keywords)
			if err != nil {
				logger.Error("Failed to search knowledge base", "cid", cid, "error", err)
				return nil, err
			}
		}
	}

	// 如果RAG服务返回为空，不写入recommendation_cache，让定时推送通过pushHotTopicsBroadcast来处理
	if len(recommendations) == 0 {
		logger.Info("No profile-based recommendations found from RAG service, not saving to cache, will be handled by hot topics broadcast", "cid", cid)
		return []models.RecommendationItem{}, nil
	}

	// 过滤特殊符号
	for i := range recommendations {
		recommendations[i].Title = utils.FilterSpecialSymbols(recommendations[i].Title)
		recommendations[i].Content = utils.FilterSpecialSymbols(recommendations[i].Content)
	}

	// 只有在有推荐内容时才保存到数据库，同时保存用户画像信息
	if err := repository.SaveRecommendations(cid, recommendations, profile); err != nil {
		logger.Error("Failed to save recommendations", "cid", cid, "error", err)
		return nil, err
	}

	logger.Info("Recommendations force generated with provided profile", "cid", cid, "count", len(recommendations))
	return recommendations, nil
}

// GetRecommendationStats 获取推荐统计信息
func GetRecommendationStats() map[string]interface{} {
	// 这里可以添加推荐系统的统计信息
	return map[string]interface{}{
		"total_users_with_recommendations": 0, // 从数据库获取
		"total_recommendations":            0,
		"last_update":                      time.Now().Format(time.RFC3339),
		"knowledge_bases":                  []string{"kb_faq"}, // 从配置获取
	}
}

// 并发生成用户推荐内容
func GenerateRecommendationsWithConcurrency(cfg *config.Config, cids []string, concurrency int) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	var mu sync.Mutex
	processed, failed, completed := 0, 0, 0

	for _, cid := range cids {
		wg.Add(1)
		semaphore <- struct{}{} // acquire semaphore

		go func(userCID string) {
			defer wg.Done()
			defer func() { <-semaphore }() // release semaphore

			_, err := GenerateRecommendationsForUser(cfg, userCID)
			mu.Lock()
			defer mu.Unlock()
			processed++
			if err != nil {
				failed++
				logger.Error("生成用户推荐内容失败", "cid", userCID, "error", err)
				return
			}
			// 不再区分created和updated，因为GenerateRecommendationsForUser内部已有完整逻辑
			completed++
			logger.Info("成功生成用户推荐内容", "cid", userCID)
		}(cid)
	}

	wg.Wait()
	logger.Info("所有用户推荐内容生成完成",
		"processed", processed,
		"completed", completed,
		"failed", failed,
	)
}
