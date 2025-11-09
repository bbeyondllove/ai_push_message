package services

import (
	"ai_push_message/config"
	"ai_push_message/logger"
	"ai_push_message/models"
	"ai_push_message/repository"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// GenerateProfileForAllUsers 为所有候选用户生成画像（并发版）
func GenerateProfileForAllUsers(cfg *config.Config) error {
	logger.Info("开始为所有候选用户生成画像")

	// 获取候选用户列表
	cids, err := repository.ListCandidateCIDs(cfg.Cron.LookbackDays)
	if err != nil {
		logger.Error("获取候选用户列表失败", "error", err)
		return err
	}
	logger.Info("找到候选用户", "count", len(cids))

	// 调用并发函数
	GenerateProfilesWithConcurrency(cfg, cids, cfg.Cron.Concurrency)

	return nil
}

// GenerateProfileForUser 为指定用户生成画像，支持合并旧画像
// 返回值: 用户画像, 是否重新生成了画像, 错误
func GenerateProfileForUser(cfg *config.Config, cid string) (*models.UserProfile, bool, error) {
	if cid == "" {
		return nil, false, fmt.Errorf("invalid CID")
	}

	lookbackDays := cfg.Cron.LookbackDays

	// 查询现有画像
	existingProfile, err := repository.GetProfile(cid)
	if err != nil && err != sql.ErrNoRows {
		return nil, false, fmt.Errorf("failed to get existing profile: %w", err)
	}

	lastTime := time.Time{}
	if existingProfile != nil {
		lastTime = existingProfile.UpdatedAt
	}

	// 聚合用户数据
	userData, err := repository.GetCombinedUserData(cid, lookbackDays, lastTime)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get user data: %w", err)
	}

	// 用户完全没有数据，不生成画像
	if !userData.HasCommunityData && !userData.HasGroupData {
		return existingProfile, false, nil // 返回 false 表示没有重新生成
	}

	// 调用 RAG 生成画像 (返回 JSON 字符串和关键词 JSON)
	newProfileJSON, newKeywordsJSON, err := fetchUserProfileFromRAGWithData(cfg, cid, userData)
	if err != nil {
		return nil, false, fmt.Errorf("failed to generate profile: %w", err)
	}

	// 如果已有旧画像，进行合并
	if existingProfile != nil {
		logger.Info("合并新旧用户画像", "user_id", cid)
		mergedProfileJSON, mergedKeywordsJSON, err := mergeProfiles(
			existingProfile.ProfileRaw,
			newProfileJSON,
			existingProfile.Keywords,
			newKeywordsJSON)
		if err != nil {
			logger.Error("合并用户画像失败", "user_id", cid, "error", err)
			return nil, false, err
		}
		newProfileJSON = mergedProfileJSON
		newKeywordsJSON = mergedKeywordsJSON
	}

	// 构造 UserProfile
	profile := &models.UserProfile{
		CID:        cid,
		ProfileRaw: newProfileJSON,
		Keywords:   newKeywordsJSON,
	}

	// Upsert 到数据库
	if err := repository.UpsertProfile(profile); err != nil {
		return nil, false, fmt.Errorf("failed to save profile: %w", err)
	}

	return profile, true, nil // 返回 true 表示重新生成了画像
}

// 并发生成用户画像
func GenerateProfilesWithConcurrency(cfg *config.Config, cids []string, concurrency int) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	var mu sync.Mutex
	processed, failed, regenerated := 0, 0, 0

	for _, cid := range cids {
		wg.Add(1)
		semaphore <- struct{}{} // acquire semaphore

		go func(userCID string) {
			defer wg.Done()
			defer func() { <-semaphore }() // release semaphore

			_, profileRegenerated, err := GenerateProfileForUser(cfg, userCID)
			mu.Lock()
			defer mu.Unlock()
			processed++
			if err != nil {
				failed++
				logger.Error("生成用户画像失败", "cid", userCID, "error", err)
				return
			}
			if profileRegenerated {
				regenerated++
				logger.Info("成功生成用户画像", "cid", userCID)
			} else {
				logger.Debug("用户画像无需更新", "cid", userCID)
			}
		}(cid)
	}

	wg.Wait()
	logger.Info("所有用户画像生成完成",
		"processed", processed,
		"regenerated", regenerated,
		"skipped", processed-regenerated-failed,
		"failed", failed,
	)
}
