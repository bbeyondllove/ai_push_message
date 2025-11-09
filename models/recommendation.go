package models

type RecommendationItem struct {
	Source        string  `json:"source"` // group_summary / rag / knowledge_base
	Title         string  `json:"title"`
	URL           string  `json:"url,omitempty"`
	Score         float64 `json:"score,omitempty"`
	RefID         string  `json:"ref_id,omitempty"`
	SearchKeyword string  `json:"search_keyword,omitempty"` // 用于搜索的关键词
	Content       string  `json:"content,omitempty"`        // 推荐内容摘要
}

type RecommendationPayload struct {
	CID   string               `json:"cid"`
	Items []RecommendationItem `json:"items"`
}
