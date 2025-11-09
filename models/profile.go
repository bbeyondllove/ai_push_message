package models

import "time"

type UserProfile struct {
	CID        string    `db:"cid" json:"cid"`
	ProfileRaw string    `db:"profile_json" json:"profile_json"` // JSON 字符串
	Keywords   string    `db:"keywords" json:"keywords"`         // JSON 字符串，如 ["DW20","合约"]
	UpdatedAt  time.Time `json:"updated_at"`
}

// WeightedKeyword 带权重的关键词
type WeightedKeyword struct {
	Keyword string  `json:"keyword"` // 关键词
	Weight  float64 `json:"weight"`  // 权重，范围0-1
}
