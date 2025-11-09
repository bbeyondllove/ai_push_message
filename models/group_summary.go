package models

import (
	"encoding/json"
)

type GroupChatSummary struct {
	ID        int64  `db:"id" json:"id"`
	GroupID   string `db:"group_id" json:"group_id"`
	GroupName string `db:"group_name" json:"group_name"`
	KeyTopics string `db:"key_topics" json:"key_topics"`
	Content   string `db:"summary_content" json:"summary_content"`
	HotTopics string `db:"hot_topics" json:"hot_topics"` // JSON格式存储的热门话题
}

// HotTopic 热门话题结构体
type HotTopic struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// ParseHotTopics 解析热门话题JSON字符串为结构体数组
func (g *GroupChatSummary) ParseHotTopics() ([]HotTopic, error) {
	if g.HotTopics == "" {
		return []HotTopic{}, nil
	}

	var topics []HotTopic
	err := json.Unmarshal([]byte(g.HotTopics), &topics)
	if err != nil {
		return []HotTopic{}, err
	}

	return topics, nil
}
