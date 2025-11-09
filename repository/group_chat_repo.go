package repository

import (
	"ai_push_message/db"
	"ai_push_message/models"
	"strconv"
	"strings"
)

// 根据关键词检索近N天群总结（key_topics/summary_content LIKE）
func SearchGroupSummariesByKeywords(keywords []string, lookbackDays, topN int) ([]models.GroupChatSummary, error) {
	if len(keywords) == 0 {
		return nil, nil
	}

	likes := make([]string, 0)
	args := make([]any, 0)
	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		likes = append(likes, "(key_topics LIKE ? OR summary_content LIKE ?)")
		arg := "%" + kw + "%"
		args = append(args, arg, arg)
	}
	if len(likes) == 0 {
		return nil, nil
	}

	where := strings.Join(likes, " OR ")
	q := "SELECT id, group_id, group_name, key_topics, summary_content FROM group_chat_summaries WHERE (" + where + ") AND created_at >= DATE_SUB(NOW(), INTERVAL ? DAY) ORDER BY created_at DESC LIMIT ?"
	args = append(args, lookbackDays, topN)

	rows, err := db.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.GroupChatSummary, 0)
	for rows.Next() {
		var g models.GroupChatSummary
		if err := rows.Scan(&g.ID, &g.GroupID, &g.GroupName, &g.KeyTopics, &g.Content); err == nil {
			out = append(out, g)
		}
	}
	return out, nil
}

// 兜底：最近的TopN
func RecentHotSummaries(topN int) ([]models.GroupChatSummary, error) {
	rows, err := db.DB.Query(`SELECT id, group_id, group_name, key_topics, summary_content FROM group_chat_summaries ORDER BY created_at DESC LIMIT ?`, topN)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.GroupChatSummary, 0)
	for rows.Next() {
		var g models.GroupChatSummary
		_ = rows.Scan(&g.ID, &g.GroupID, &g.GroupName, &g.KeyTopics, &g.Content)
		out = append(out, g)
	}
	return out, nil
}

// GetHotTopicsFromGroupSummaries 从群聊总结中获取热门话题作为推荐内容（只获取前一天的）
func GetHotTopicsFromGroupSummaries() ([]models.HotTopic, error) {
	rows, err := db.DB.Query(`
		SELECT hot_topics 
		FROM group_chat_summaries 
		WHERE hot_topics IS NOT NULL 
			AND hot_topics != '' 
			AND JSON_VALID(hot_topics) = 1
			AND DATE(created_at) = DATE_SUB(CURDATE(), INTERVAL 1 DAY)
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allTopics []models.HotTopic
	for rows.Next() {
		var hotTopicsJSON string
		if err := rows.Scan(&hotTopicsJSON); err != nil {
			continue
		}

		// 创建临时的GroupChatSummary来使用ParseHotTopics方法
		tempSummary := models.GroupChatSummary{
			HotTopics: hotTopicsJSON,
		}

		topics, err := tempSummary.ParseHotTopics()
		if err != nil {
			continue
		}

		allTopics = append(allTopics, topics...)
	}

	return allTopics, nil
}

func IdToString(id int64) string {
	return strconv.FormatInt(id, 10)
}
