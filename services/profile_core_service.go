package services

import (
	"ai_push_message/config"
	"ai_push_message/logger"
	"ai_push_message/models"
	"ai_push_message/repository"
	"ai_push_message/utils"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// fetchUserProfileFromRAGWithData 使用用户数据获取用户画像
func fetchUserProfileFromRAGWithData(cfg *config.Config, cid string, userData *repository.CombinedUserData) (string, string, error) {
	// 构建用户数据分析提示词
	prompt := buildUserAnalysisPrompt(cid, userData)

	// 调用LLM分析用户画像
	profileData, keywords, err := callLLMForUserProfile(cfg, prompt)
	if err != nil {
		logger.Error("LLM分析失败", "user_id", cid, "error", err)
		// 降级到基础分析
		return fallbackProfileGeneration(cid, userData)
	}

	return profileData, keywords, nil
}

// mergeProfiles 合并新旧用户画像
func mergeProfiles(oldProfileJSON, newProfileJSON, oldKeywordsJSON, newKeywordsJSON string) (string, string, error) {
	var oldProfile, newProfile map[string]interface{}
	var oldKeywords, newKeywords []string

	// 解析旧画像
	if err := json.Unmarshal([]byte(oldProfileJSON), &oldProfile); err != nil {
		return "", "", fmt.Errorf("解析旧画像失败: %v", err)
	}

	// 解析新画像
	if err := json.Unmarshal([]byte(newProfileJSON), &newProfile); err != nil {
		return "", "", fmt.Errorf("解析新画像失败: %v", err)
	}

	// 解析旧关键词
	if err := json.Unmarshal([]byte(oldKeywordsJSON), &oldKeywords); err != nil {
		return "", "", fmt.Errorf("解析旧关键词失败: %v", err)
	}

	// 解析新关键词
	if err := json.Unmarshal([]byte(newKeywordsJSON), &newKeywords); err != nil {
		return "", "", fmt.Errorf("解析新关键词失败: %v", err)
	}

	// 合并兴趣
	var mergedInterests []string
	if oldInts, ok := oldProfile["interests"].([]interface{}); ok {
		for _, item := range oldInts {
			if interest, ok := item.(string); ok {
				mergedInterests = append(mergedInterests, interest)
			}
		}
	}
	if newInts, ok := newProfile["interests"].([]interface{}); ok {
		for _, item := range newInts {
			if interest, ok := item.(string); ok {
				mergedInterests = append(mergedInterests, interest)
			}
		}
	}
	mergedInterests = utils.DeduplicateSlice(mergedInterests)

	// 合并关键词权重
	allKeywords := make(map[string]float64)

	// 处理旧画像中的关键词权重
	if oldWk, ok := oldProfile["weighted_keywords"].([]interface{}); ok {
		for _, item := range oldWk {
			if kwObj, ok := item.(map[string]interface{}); ok {
				keyword, _ := kwObj["keyword"].(string)
				weight, _ := kwObj["weight"].(float64)
				if keyword != "" {
					allKeywords[keyword] = weight
				}
			}
		}
	}

	// 处理新画像中的关键词权重，取最高权重
	if newWk, ok := newProfile["weighted_keywords"].([]interface{}); ok {
		for _, item := range newWk {
			if kwObj, ok := item.(map[string]interface{}); ok {
				keyword, _ := kwObj["keyword"].(string)
				weight, _ := kwObj["weight"].(float64)
				if keyword != "" {
					if existingWeight, exists := allKeywords[keyword]; exists {
						// 取最高权重
						if weight > existingWeight {
							allKeywords[keyword] = weight
						}
					} else {
						allKeywords[keyword] = weight
					}
				}
			}
		}
	}

	// 构建合并后的加权关键词
	var mergedWeightedKeywords []models.WeightedKeyword
	for keyword, weight := range allKeywords {
		mergedWeightedKeywords = append(mergedWeightedKeywords, models.WeightedKeyword{
			Keyword: keyword,
			Weight:  weight,
		})
	}

	// 按权重排序
	sort.Slice(mergedWeightedKeywords, func(i, j int) bool {
		return mergedWeightedKeywords[i].Weight > mergedWeightedKeywords[j].Weight
	})

	// 合并关键词列表
	mergedKeywords := append(oldKeywords, newKeywords...)
	mergedKeywords = utils.DeduplicateSlice(mergedKeywords)

	// 确定活跃度，取最高级别
	activityLevel := "low"
	if oldLevel, ok := oldProfile["activity_level"].(string); ok {
		activityLevel = oldLevel
	}
	if newLevel, ok := newProfile["activity_level"].(string); ok {
		if newLevel == "high" || (newLevel == "medium" && activityLevel == "low") {
			activityLevel = newLevel
		}
	}

	// 确定用户类型，优先使用新画像中的类型
	userType := ""
	if oldType, ok := oldProfile["user_type"].(string); ok {
		userType = oldType
	}
	if newType, ok := newProfile["user_type"].(string); ok && newType != "" {
		userType = newType
	}

	// 构建最终合并画像
	mergedProfile := map[string]interface{}{
		"interests":         mergedInterests,
		"weighted_keywords": mergedWeightedKeywords,
		"activity_level":    activityLevel,
		"user_type":         userType,
		"updated_at":        time.Now().Format(time.RFC3339),
	}

	// 确保数据一致性：如果有interests但没有weighted_keywords，从 interests 生成
	if len(mergedInterests) > 0 && len(mergedWeightedKeywords) == 0 {
		for i, interest := range mergedInterests {
			// 根据位置分配权重，首个兴趣权重最高
			weight := 0.9 - float64(i)*0.1
			if weight < 0.1 {
				weight = 0.1
			}
			mergedWeightedKeywords = append(mergedWeightedKeywords, models.WeightedKeyword{
				Keyword: interest,
				Weight:  weight,
			})
			mergedKeywords = append(mergedKeywords, interest)
		}
		mergedProfile["weighted_keywords"] = mergedWeightedKeywords
		logger.Info("合并时从兴趣生成加权关键词", "count", len(mergedWeightedKeywords))
	}

	// 反之，如果没有interests但有weighted_keywords，从 weighted_keywords 生成 interests
	if len(mergedInterests) == 0 && len(mergedWeightedKeywords) > 0 {
		for _, wk := range mergedWeightedKeywords {
			mergedInterests = append(mergedInterests, wk.Keyword)
		}
		mergedProfile["interests"] = mergedInterests
		logger.Info("合并时从加权关键词生成兴趣", "count", len(mergedInterests))
	}

	// 保留其他可能的字段
	for key, value := range oldProfile {
		if _, exists := mergedProfile[key]; !exists {
			mergedProfile[key] = value
		}
	}
	for key, value := range newProfile {
		if _, exists := mergedProfile[key]; !exists {
			mergedProfile[key] = value
		}
	}

	// 序列化合并后的画像和关键词
	mergedProfileJSON, _ := json.Marshal(mergedProfile)
	mergedKeywordsJSON, _ := json.Marshal(mergedKeywords)

	return string(mergedProfileJSON), string(mergedKeywordsJSON), nil
}

// fallbackProfileGeneration 降级的画像生成方法
func fallbackProfileGeneration(cid string, userData *repository.CombinedUserData) (string, string, error) {
	keywordFrequency := make(map[string]int)
	allContent := make([]string, 0)

	// Add community posts content
	for _, content := range userData.CommunityPosts {
		allContent = append(allContent, content)
		keywords := extractKeywordsFromContent(content)
		for _, kw := range keywords {
			keywordFrequency[kw]++
		}
	}

	// Add group messages content
	for _, message := range userData.GroupMessages {
		allContent = append(allContent, message)
		keywords := extractKeywordsFromContent(message)
		for _, kw := range keywords {
			keywordFrequency[kw]++
		}
	}

	// Add group interests
	for _, interest := range userData.GroupInterests {
		keywordFrequency[interest]++
	}

	// 计算权重并排序
	type keywordWeight struct {
		keyword string
		weight  float64
	}

	var weightedKeywords []keywordWeight
	var maxFreq int = 1

	// 找出最大频率
	for _, freq := range keywordFrequency {
		if freq > maxFreq {
			maxFreq = freq
		}
	}

	// 计算权重
	for kw, freq := range keywordFrequency {
		weight := float64(freq) / float64(maxFreq)
		weightedKeywords = append(weightedKeywords, keywordWeight{
			keyword: kw,
			weight:  weight,
		})
	}

	// 按权重降序排序
	sort.Slice(weightedKeywords, func(i, j int) bool {
		return weightedKeywords[i].weight > weightedKeywords[j].weight
	})

	// 转换为模型格式
	var modelWeightedKeywords []models.WeightedKeyword
	var orderedKeywords []string

	for _, kw := range weightedKeywords {
		modelWeightedKeywords = append(modelWeightedKeywords, models.WeightedKeyword{
			Keyword: kw.keyword,
			Weight:  kw.weight,
		})
		orderedKeywords = append(orderedKeywords, kw.keyword)
	}

	// Generate enhanced profile based on combined data
	enhancedProfile := map[string]interface{}{
		"user_id": cid,
		"data_sources": map[string]bool{
			"community_posts": userData.HasCommunityData,
			"group_activity":  userData.HasGroupData,
		},
		"content_sources": map[string]interface{}{
			"community_posts_count": len(userData.CommunityPosts),
			"group_messages_count":  len(userData.GroupMessages),
			"active_groups":         userData.ActiveGroups,
		},
		"interests":         orderedKeywords, // 使用有序关键词作为兴趣
		"weighted_keywords": modelWeightedKeywords,
		"activity_level":    determineActivityLevel(userData),
		"updated_at":        time.Now().Format(time.RFC3339),
	}

	// 确保数据一致性：如果没有interests但有weighted_keywords，从 weighted_keywords 生成 interests
	if len(orderedKeywords) == 0 && len(modelWeightedKeywords) > 0 {
		var interests []string
		for _, wk := range modelWeightedKeywords {
			interests = append(interests, wk.Keyword)
		}
		enhancedProfile["interests"] = interests
		orderedKeywords = interests // 更新 orderedKeywords 以便后续使用
		logger.Info("fallback中从加权关键词生成兴趣", "count", len(interests))
	}

	profileJSON, _ := json.Marshal(enhancedProfile)
	keywordsJSON, _ := json.Marshal(orderedKeywords)

	return string(profileJSON), string(keywordsJSON), nil
}

// extractKeywordsFromContent extracts key topics from content (simplified)
func extractKeywordsFromContent(content string) []string {
	// In a real implementation, this would use NLP techniques
	// For now, return simple keyword extraction
	keywords := []string{
		"区块链", "人工智能", "投资", "教育", "技术", "金融",
		"加密货币", "DW20", "无链", "数字货币", "创新",
		"比特币", "交易", "钱包", "质押", "挖矿", "去中心化",
		"智能合约", "NFT", "元宇宙", "Web3", "DAO",
	}

	found := make([]string, 0)
	contentLower := strings.ToLower(content)
	for _, kw := range keywords {
		if strings.Contains(contentLower, strings.ToLower(kw)) {
			found = append(found, kw)
		}
	}

	return found
}

// determineActivityLevel determines user activity based on data sources
func determineActivityLevel(data *repository.CombinedUserData) string {
	postCount := len(data.CommunityPosts)
	messageCount := len(data.GroupMessages)
	groupCount := len(data.ActiveGroups)

	totalActivity := postCount + messageCount + groupCount

	if totalActivity > 10 {
		return "high"
	} else if totalActivity > 3 {
		return "medium"
	} else if totalActivity > 0 {
		return "low"
	}
	return "minimal"
}
