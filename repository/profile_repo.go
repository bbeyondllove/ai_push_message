package repository

import (
	"ai_push_message/db"
	"ai_push_message/logger"
	"ai_push_message/models"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// =====================
// 通用工具函数
// =====================

// queryStrings 执行查询并返回字符串结果列表
func queryStrings(query string, args ...interface{}) ([]string, error) {
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]string, 0)
	for rows.Next() {
		var val sql.NullString
		if err := rows.Scan(&val); err == nil && val.Valid {
			s := strings.TrimSpace(val.String)
			if s != "" {
				results = append(results, s)
			}
		}
	}
	return results, nil
}

// exists 执行 COUNT(1) 查询并返回是否存在数据
func exists(query string, args ...interface{}) (bool, error) {
	var count int
	err := db.DB.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// =====================
// 用户画像相关
// =====================

func GetProfile(cid string) (*models.UserProfile, error) {
	row := db.DB.QueryRow(`SELECT cid, profile_json, keywords, updated_at FROM user_profiles WHERE cid=?`, cid)
	p := &models.UserProfile{}
	if err := row.Scan(&p.CID, &p.ProfileRaw, &p.Keywords, &p.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return p, nil
}

func UpsertProfile(p *models.UserProfile) error {
	_, err := db.DB.Exec(`
        INSERT INTO user_profiles (cid, profile_json, keywords, updated_at, created_at)
        VALUES (?, ?, ?, NOW(), NOW())
        ON DUPLICATE KEY UPDATE profile_json=VALUES(profile_json), keywords=VALUES(keywords), updated_at=NOW()
    `, p.CID, p.ProfileRaw, p.Keywords)
	return err
}

// =====================
// 候选用户列表
// =====================

// ListCandidateCIDs returns user CIDs for profile generation
func ListCandidateCIDs(lookbackDays int) ([]string, error) {
	cids := make([]string, 0)
	seen := make(map[string]bool)

	localQueries := []string{
		`SELECT DISTINCT sender_id AS cid FROM group_chat_messages WHERE sender_id IS NOT NULL AND sender_id != ''`,
		`SELECT DISTINCT cid FROM user_community_posting_record WHERE cid IS NOT NULL AND cid != ''`,
	}

	for _, query := range localQueries {
		rows, err := db.DB.Query(query)
		if err != nil {
			continue // 表可能不存在
		}
		defer rows.Close()

		for rows.Next() {
			var cid string
			if err := rows.Scan(&cid); err != nil {
				continue
			}
			cid = strings.TrimSpace(cid)
			if cid != "" && !seen[cid] {
				cids = append(cids, cid)
				seen[cid] = true
			}
		}
	}

	return cids, nil
}

// =====================
// 用户内容提取
// =====================

// GetUserContentFromSimiCommunity returns user-generated content from simi_community_history
func GetUserContentFromSimiCommunity(cid string, lookbackDays int) ([]string, error) {
	if cid == "" {
		return nil, nil
	}
	query := `
        SELECT a.article_text 
        FROM simi_community_history a
        JOIN user_community_posting_record b ON a.article_id = b.article_id
        WHERE b.cid = ? 
          AND a.created_at >= DATE_SUB(NOW(), INTERVAL ? DAY)
          AND a.article_text IS NOT NULL AND a.article_text != ''`
	return queryStrings(query, cid, lookbackDays)
}

// ExtractTopicsFromMessage extracts topics from chat messages
func ExtractTopicsFromMessage(content, title string) []string {
	topics := make([]string, 0)
	fullText := strings.ToLower(content + " " + title)

	keywords := []string{
		"区块链", "无链", "dw20", "质押", "交易", "投资",
		"群聊", "钱包", "注册", "奖励", "推荐", "上所",
		"技术", "讨论", "问题", "解答", "官方群", "新手群",
	}
	for _, keyword := range keywords {
		if strings.Contains(fullText, strings.ToLower(keyword)) {
			topics = append(topics, keyword)
		}
	}
	return topics
}

// ExtractTopicsFromMessages 批量提取话题，去重
func ExtractTopicsFromMessages(messages []string, titles []string) []string {
	seen := make(map[string]bool)
	topics := make([]string, 0)
	for i, msg := range messages {
		t := ""
		if i < len(titles) {
			t = titles[i]
		}
		for _, topic := range ExtractTopicsFromMessage(msg, t) {
			if !seen[topic] {
				topics = append(topics, topic)
				seen[topic] = true
			}
		}
	}
	return topics
}

// =====================
// 用户数据检查
// =====================

func HasUserData(cid string) (bool, error) {
	if cid == "" {
		return false, nil
	}
	q1 := `SELECT COUNT(1) FROM group_chat_messages WHERE sender_id = ?`
	if ok, _ := exists(q1, cid); ok {
		return true, nil
	}
	q2 := `SELECT COUNT(1) 
	       FROM simi_community_history a 
	       JOIN user_community_posting_record b ON a.article_id = b.article_id
	       WHERE b.cid = ?`
	return exists(q2, cid)
}

func HasNewerDataThan(cid string, lastUpdateTime time.Time) (bool, error) {
	if cid == "" {
		return false, errors.New("invalid CID")
	}
	q1 := `SELECT COUNT(1) FROM group_chat_messages WHERE sender_id = ? AND message_time > ?`
	if ok, _ := exists(q1, cid, lastUpdateTime); ok {
		return true, nil
	}
	q2 := `SELECT COUNT(1)
	       FROM simi_community_history a 
	       JOIN user_community_posting_record b ON a.article_id = b.article_id
	       WHERE b.cid = ? AND a.created_at > ?`
	return exists(q2, cid, lastUpdateTime)
}

// =====================
// 用户数据聚合
// =====================

type CombinedUserData struct {
	CID              string
	CommunityPosts   []string
	GroupMessages    []string
	ActiveGroups     []string
	GroupInterests   []string
	SenderRole       string
	MostRecent       time.Time
	HasCommunityData bool
	HasGroupData     bool
}

func GetCombinedUserData(cid string, lookbackDays int, lastTime time.Time) (*CombinedUserData, error) {
	if cid == "" {
		return nil, errors.New("invalid CID")
	}

	data := &CombinedUserData{
		CID:            cid,
		CommunityPosts: make([]string, 0),
		GroupMessages:  make([]string, 0),
		ActiveGroups:   make([]string, 0),
		GroupInterests: make([]string, 0),
	}

	// -----------------------
	// 社区帖子
	// -----------------------
	// 使用参数化查询避免SQL注入
	queryCommunity := `SELECT a.article_text 
		 FROM simi_community_history a
		 JOIN user_community_posting_record b ON a.article_id = b.article_id
		 WHERE b.cid = ? 
		   AND STR_TO_DATE(a.create_time, '%Y-%m-%d %H:%i:%s') >= DATE_SUB(NOW(), INTERVAL ? DAY)
		   AND STR_TO_DATE(a.create_time, '%Y-%m-%d %H:%i:%s') > ?
		   AND a.article_text IS NOT NULL AND a.article_text != ''`

	rows, err := db.DB.Query(queryCommunity, cid, lookbackDays, lastTime)
	if err != nil {
		logger.Error("Failed to query community posts", "error", err)
		return data, nil // 返回部分数据而不是错误
	}
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()

	posts := make([]string, 0)
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err == nil && text != "" {
			posts = append(posts, strings.TrimSpace(text))
		}
	}

	if len(posts) > 0 {
		data.CommunityPosts = posts
		data.HasCommunityData = true
	}

	// -----------------------
	// 群聊消息 & 群组
	// -----------------------
	// 使用参数化查询避免SQL注入
	queryGroup := `SELECT content, title, group_name
		 FROM group_chat_messages 
		 WHERE sender_id = ? 
		   AND message_time >= DATE_SUB(NOW(), INTERVAL ? DAY)
		   AND message_time > ?`

	rows, err = db.DB.Query(queryGroup, cid, lookbackDays, lastTime)
	if err != nil {
		logger.Error("Failed to query group messages", "error", err)
		return data, nil // 返回部分数据而不是错误
	}
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()

	messages, titles, groups := make([]string, 0), make([]string, 0), make([]string, 0)
	groupSeen := make(map[string]bool)

	for rows.Next() {
		var content, title, groupName sql.NullString
		if err := rows.Scan(&content, &title, &groupName); err == nil {
			if content.Valid && content.String != "" {
				messages = append(messages, strings.TrimSpace(content.String))
			}
			if title.Valid {
				titles = append(titles, strings.TrimSpace(title.String))
			}
			if groupName.Valid && groupName.String != "" {
				gName := strings.TrimSpace(groupName.String)
				if !groupSeen[gName] {
					groups = append(groups, gName)
					groupSeen[gName] = true
				}
			}
		}
	}

	if len(messages) > 0 {
		data.GroupMessages = messages
		data.GroupInterests = ExtractTopicsFromMessages(messages, titles)
		data.HasGroupData = true
	}
	if len(groups) > 0 {
		data.ActiveGroups = groups
		data.HasGroupData = true
	}

	return data, nil
}
