package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"ai_push_message/config"
	"ai_push_message/logger"
	"ai_push_message/models"
	"ai_push_message/repository"
	"ai_push_message/utils"
)

func aggregate(itemsA, itemsB []models.RecommendationItem, topN int) []models.RecommendationItem {
	m := map[string]models.RecommendationItem{}
	for _, it := range append(itemsA, itemsB...) {
		key := it.Source + "|" + it.RefID + "|" + it.Title
		if old, ok := m[key]; ok {
			if it.Score > old.Score {
				m[key] = it
			}
		} else {
			m[key] = it
		}
	}
	arr := make([]models.RecommendationItem, 0, len(m))
	for _, v := range m {
		arr = append(arr, v)
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].Score > arr[j].Score })
	if topN > 0 && len(arr) > topN {
		arr = arr[:topN]
	}
	return arr
}

// RecommendationPushPayload 表示推送到外部API的推荐内容数据
type RecommendationPushPayload struct {
	CID  string          `json:"cid,omitempty"`
	Tags []TagPushFormat `json:"tags"`
}

// TagPushFormat 表示推送给外部API的标签格式
type TagPushFormat struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// 通过HTTP推送内容给第三方服务器
func pushViaHTTP(cfg *config.Config, cid string, items []models.RecommendationItem) bool {
	// 将RecommendationItem转换为TagPushFormat（数据已在保存时过滤过特殊符号）
	tags := make([]TagPushFormat, 0, len(items))
	for _, item := range items {
		tags = append(tags, TagPushFormat{
			Title:   item.Title,
			Content: item.Content,
		})
	}

	// 构建推送数据
	payload := RecommendationPushPayload{
		Tags: tags,
	}

	// 只有当cid非空时才设置CID字段
	if cid != "" {
		payload.CID = cid
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger.Error("序列化推荐内容数据失败", "error", err, "user_id", cid)
		return false
	}

	// 记录推送数据的详细日志
	prettyJSON, _ := json.MarshalIndent(payload, "", "  ")
	logger.Info("准备推送的数据（请求体）", "user_id", cid, "payload", string(prettyJSON))

	// 准备HTTP请求
	pushURL := cfg.ExternalAPI.TagPushURL
	req, err := http.NewRequest("POST", pushURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("创建HTTP请求失败", "error", err, "user_id", cid)
		return false
	}

	// 设置请求头
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	timestampStr := strconv.FormatInt(timestamp, 10)

	// 获取时间戳后4位
	lastFourDigits := timestampStr[len(timestampStr)-4:]

	// 直接从环境变量中读取API密钥，而不是从配置对象中获取
	apiKey := os.Getenv("EXTERNAL_API_KEY")
	if apiKey == "" {
		logger.Warn("环境变量EXTERNAL_API_KEY未设置", "apiKey", apiKey)
	} else {
		logger.Info("从环境变量读取的API密钥", "apiKey", apiKey)
	}

	authorization := utils.CalculateAuthorizationHeader(apiKey, lastFourDigits)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("timestamp", timestampStr)
	req.Header.Set("Authorization", authorization)
	req.Header.Set("apiKey", apiKey)

	// 记录请求头和请求体信息
	logger.Info("HTTP推送请求信息",
		"url", pushURL,
		"timestamp", timestampStr,
		"timestamp_last_4", lastFourDigits,
		"apiKey", apiKey,
		"apiKey+timestamp_last_4", apiKey+lastFourDigits,
		"Authorization", authorization,
		"body", string(jsonData))

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("发送推荐内容推送请求失败", "error", err, "user_id", cid)
		return false
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		logger.Error("推荐内容推送请求返回非200状态码", "status_code", resp.StatusCode, "user_id", cid)
		return false
	}

	// 解析响应
	var result struct {
		ErrCode int    `json:"errCode"`
		Msg     string `json:"msg"`
		Success bool   `json:"success"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Error("解析推荐内容推送响应失败", "error", err, "user_id", cid)
		return false
	}

	if !result.Success || result.ErrCode != 200 {
		logger.Error("推荐内容推送失败", "error_code", result.ErrCode, "message", result.Msg, "user_id", cid)
		return false
	}

	logger.Info("成功通过HTTP推送推荐内容", "count", len(items), "user_id", cid)
	return true
}

// PushForCID 为指定用户推送推荐内容，不考虑pushed标志
func PushForCID(cfg *config.Config, cid string) error {
	start := time.Now()

	// 直接从数据库获取用户的推荐内容
	recommendations, err := repository.GetRecommendations(cid)
	if err != nil {
		logger.Info("用户没有推荐内容，跳过推送", "user_id", cid, "error", err.Error())
		return nil
	}

	// 检查推荐内容是否为空
	if len(recommendations) == 0 {
		logger.Info("用户没有推荐内容，跳过推送", "user_id", cid)
		return nil
	}

	// 通过HTTP推送内容
	pushOk := pushViaHTTP(cfg, cid, recommendations)

	// 不再标记为已推送，API接口推送不受pushed标志限制

	logger.Info("推送完成",
		"user_id", cid,
		"items", len(recommendations),
		"method", "http",
		"success", pushOk,
		"cost", time.Since(start).String())
	return nil
}

// PushAll 推送所有用户的推荐内容，不考虑pushed标志
func PushAll(cfg *config.Config) error {
	logger.Info("开始推送所有用户的推荐内容")

	// 直接从数据库获取所有推荐内容
	recommendations, err := repository.GetAllRecommendations()
	if err != nil {
		logger.Error("获取所有推荐内容失败", "error", err)
		return err
	}

	logger.Info("找到有推荐内容的用户", "count", len(recommendations))

	// 使用并发推送
	successCount, failCount := PushRecommendationsWithConcurrency(cfg, recommendations)

	//发送热门话题群发消息

	logger.Info("开始发送热门话题群发消息")
	if err := pushHotTopicsBroadcast(cfg); err != nil {
		logger.Error("发送热门话题群发消息失败", "error", err)
		failCount++
	} else {
		logger.Info("热门话题群发消息发送成功")
		successCount++
	}

	logger.Info("推送完成", "success", successCount, "failed", failCount)
	return nil
}

// pushHotTopicsBroadcast 发送热门话题群发消息（cid=""）
func pushHotTopicsBroadcast(cfg *config.Config) error {
	logger.Info("开始获取前一天的热门话题用于群发")

	// 获取热门话题作为推荐内容
	hotTopicsRecommendations, err := GetHotTopicsAsRecommendations(cfg)
	if err != nil {
		logger.Error("获取热门话题失败", "error", err)
		return err
	}

	if len(hotTopicsRecommendations) == 0 {
		logger.Info("没有找到前一天的热门话题，跳过群发")
		return nil
	}

	logger.Info("获取到热门话题", "count", len(hotTopicsRecommendations))

	// 使用空的cid进行群发
	pushOk := pushViaHTTP(cfg, "", hotTopicsRecommendations)
	if !pushOk {
		return fmt.Errorf("热门话题群发消息发送失败")
	}

	logger.Info("热门话题群发消息发送成功", "topics_count", len(hotTopicsRecommendations))
	return nil
}

// PushRecommendationsWithConcurrency 并发推送用户推荐内容
func PushRecommendationsWithConcurrency(cfg *config.Config, recommendations map[string][]models.RecommendationItem) (int, int) {
	// 获取推送并发数配置
	pushConcurrency := cfg.Cron.PushConcurrency

	// 转换为列表便于并发处理
	type pushItem struct {
		cid   string
		items []models.RecommendationItem
	}

	var pushList []pushItem
	for cid, items := range recommendations {
		if len(items) > 0 {
			pushList = append(pushList, pushItem{cid: cid, items: items})
		}
	}

	logger.Info("开始并发推送", "total_users", len(pushList), "concurrency", pushConcurrency)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, pushConcurrency)

	var mu sync.Mutex
	var successCount, failCount int

	for _, item := range pushList {
		wg.Add(1)
		semaphore <- struct{}{} // acquire semaphore

		go func(pushData pushItem) {
			defer wg.Done()
			defer func() { <-semaphore }() // release semaphore

			// 通过HTTP推送内容
			pushOk := pushViaHTTP(cfg, pushData.cid, pushData.items)

			mu.Lock()
			if pushOk {
				successCount++
				logger.Info("用户推送成功", "cid", pushData.cid, "items_count", len(pushData.items))
			} else {
				failCount++
				logger.Error("用户推送失败", "cid", pushData.cid, "items_count", len(pushData.items))
			}
			mu.Unlock()
		}(item)
	}

	wg.Wait()
	logger.Info("并发推送完成", "success", successCount, "failed", failCount, "concurrency", pushConcurrency)

	return successCount, failCount
}
