package services

import (
	"ai_push_message/models"
	"ai_push_message/repository"
	"database/sql"
	"encoding/json"
)

func LoadUserProfile(cid string) (*models.UserProfile, error) {
	p, err := repository.GetProfile(cid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func ExtractKeywords(p *models.UserProfile) []string {
	if p == nil || p.Keywords == "" {
		return nil
	}
	var arr []string
	_ = json.Unmarshal([]byte(p.Keywords), &arr)
	return arr
}

// ValidateUserProfile 检查用户是否有画像
func ValidateUserProfile(cid string) (bool, error) {
	profile, err := LoadUserProfile(cid)
	if err != nil {
		return false, err
	}
	return profile != nil, nil
}
