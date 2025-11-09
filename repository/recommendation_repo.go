package repository

import (
	"ai_push_message/db"
	"ai_push_message/models"
	"encoding/json"
	"strings"
)

func SaveRecommendationCache(cid string, items []models.RecommendationItem, algo string, userProfile *models.UserProfile) error {
	b, _ := json.Marshal(map[string]any{"recommendations": items})

	// 序列化用户画像信息
	var userProfileJSON string
	if userProfile != nil {
		if profileBytes, err := json.Marshal(userProfile); err == nil {
			userProfileJSON = string(profileBytes)
		}
	}

	// 检查是否已存在推荐内容
	var count int
	err := db.DB.QueryRow(`SELECT COUNT(*) FROM recommendation_cache WHERE cid = ?`, cid).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		// 更新现有记录，同时更新用户画像信息
		_, err = db.DB.Exec(`
			UPDATE recommendation_cache 
			SET recommendations = CAST(? AS JSON), 
				user_profile = CASE WHEN ? != '' THEN CAST(? AS JSON) ELSE user_profile END,
				algorithm = ?, 
				generated_at = NOW()
			WHERE cid = ?
		`, string(b), userProfileJSON, userProfileJSON, algo, cid)
	} else {
		// 插入新记录，包含用户画像信息
		_, err = db.DB.Exec(`
			INSERT INTO recommendation_cache (cid, recommendations, user_profile, algorithm, generated_at, pushed)
			VALUES (?, CAST(? AS JSON), CASE WHEN ? != '' THEN CAST(? AS JSON) ELSE NULL END, ?, NOW(), 0)
		`, cid, string(b), userProfileJSON, userProfileJSON, algo)
	}

	return err
}

func MarkPushed(cid string) error {
	_, err := db.DB.Exec(`UPDATE recommendation_cache SET pushed=1, pushed_at=NOW() WHERE cid=? AND pushed=0`, cid)
	return err
}

// SaveRecommendations 保存用户推荐结果和用户画像
func SaveRecommendations(cid string, items []models.RecommendationItem, userProfile *models.UserProfile) error {
	return SaveRecommendationCache(cid, items, "profile_based", userProfile)
}

// GetRecommendations 获取用户推荐内容
func GetRecommendations(cid string) ([]models.RecommendationItem, error) {
	var recommendationsJSON string
	err := db.DB.QueryRow(`
		SELECT recommendations 
		FROM recommendation_cache 
		WHERE cid = ? 
		ORDER BY generated_at DESC 
		LIMIT 1
	`, cid).Scan(&recommendationsJSON)

	if err != nil {
		return nil, err
	}

	var result struct {
		Recommendations []models.RecommendationItem `json:"recommendations"`
	}

	if err := json.Unmarshal([]byte(recommendationsJSON), &result); err != nil {
		return nil, err
	}

	return result.Recommendations, nil
}

// ListUsersWithProfiles 获取所有有画像的用户列表
func ListUsersWithProfiles() ([]string, error) {
	rows, err := db.DB.Query(`
		SELECT DISTINCT cid 
		FROM user_profiles 
		WHERE profile_json IS NOT NULL 
		AND profile_json != '' 
		AND keywords IS NOT NULL 
		AND keywords != ''
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cids []string
	for rows.Next() {
		var cid string
		if err := rows.Scan(&cid); err != nil {
			continue
		}
		cids = append(cids, cid)
	}

	return cids, nil
}

// GetAllUnpushedRecommendations 获取所有未推送的推荐内容
func GetAllUnpushedRecommendations() (map[string][]models.RecommendationItem, error) {
	rows, err := db.DB.Query(`
		SELECT cid, recommendations 
		FROM recommendation_cache 
		WHERE recommendations IS NOT NULL 
		AND recommendations != '' 
		AND pushed = 0
		ORDER BY generated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]models.RecommendationItem)
	for rows.Next() {
		var cid, recommendationsJSON string
		if err := rows.Scan(&cid, &recommendationsJSON); err != nil {
			continue
		}

		var recData struct {
			Recommendations []models.RecommendationItem `json:"recommendations"`
		}

		if err := json.Unmarshal([]byte(recommendationsJSON), &recData); err != nil {
			continue
		}

		if len(recData.Recommendations) > 0 {
			result[cid] = recData.Recommendations
		}
	}

	return result, nil
}

// GetAllRecommendations 获取所有推荐内容（不考虑pushed状态）
func GetAllRecommendations() (map[string][]models.RecommendationItem, error) {
	rows, err := db.DB.Query(`
		SELECT cid, recommendations 
		FROM recommendation_cache 
		WHERE recommendations IS NOT NULL 
		AND recommendations != '' 
		ORDER BY generated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]models.RecommendationItem)
	for rows.Next() {
		var cid, recommendationsJSON string
		if err := rows.Scan(&cid, &recommendationsJSON); err != nil {
			continue
		}

		var recData struct {
			Recommendations []models.RecommendationItem `json:"recommendations"`
		}

		if err := json.Unmarshal([]byte(recommendationsJSON), &recData); err != nil {
			continue
		}

		if len(recData.Recommendations) > 0 {
			result[cid] = recData.Recommendations
		}
	}

	return result, nil
}

// GetUserInterestsFromGroupMessages 从群聊消息中提取用户兴趣
func GetUserInterestsFromGroupMessages(cid string, lookbackDays int) ([]string, error) {
	interests := make([]string, 0)

	if cid == "" {
		return interests, nil
	}

	query := `SELECT content, title, group_name
              FROM group_chat_messages 
              WHERE sender_id = ? 
                  AND message_time >= DATE(DATE_SUB(NOW(), INTERVAL ? DAY))
                  AND content IS NOT NULL AND content != ''
              ORDER BY message_time DESC`

	rows, err := db.DB.Query(query, cid, lookbackDays)
	if err != nil {
		return interests, nil
	}
	defer rows.Close()

	seen := make(map[string]bool)
	for rows.Next() {
		var content, title, groupName string
		if err := rows.Scan(&content, &title, &groupName); err == nil {
			// Extract topics from message content
			topics := extractTopicsFromRecommendation(content, title)
			for _, topic := range topics {
				if topic != "" && !seen[topic] {
					interests = append(interests, topic)
					seen[topic] = true
				}
			}
		}
	}

	return interests, nil
}

// extractTopicsFromRecommendation 从消息中提取话题（推荐系统专用）
func extractTopicsFromRecommendation(content, title string) []string {
	topics := make([]string, 0)

	// 合并内容和标题进行话题提取
	fullText := strings.ToLower(content + " " + title)

	// 使用关键词模式提取话题
	keywords := []string{
		"区块链", "无链", "dw20", "质押", "交易", "投资",
		"群聊", "钱包", "注册", "奖励", "推荐", "上所",
		"技术", "讨论", "问题", "解答", "官方群", "新手群",
		"比特币", "以太坊", "数字货币", "加密货币", "挖矿",
		"去中心化", "智能合约", "NFT", "DeFi", "Web3",
	}

	for _, keyword := range keywords {
		if strings.Contains(fullText, strings.ToLower(keyword)) {
			topics = append(topics, keyword)
		}
	}

	return topics
}

// HasUsersWithoutRecommendations 检查是否有user_profiles中的cid不在recommendation_cache中
func HasUsersWithoutRecommendations() (bool, error) {
	var count int
	err := db.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM user_profiles up 
		LEFT JOIN recommendation_cache rc ON up.cid = rc.cid 
		WHERE up.profile_json IS NOT NULL 
			AND up.profile_json != '' 
			AND (rc.cid IS NULL OR rc.recommendations IS NULL OR rc.recommendations = '')
	`).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
