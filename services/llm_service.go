package services

import (
	"ai_push_message/config"
	"ai_push_message/logger"
	"ai_push_message/models"
	"ai_push_message/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// 定义SiliconFlow API请求和响应结构
type siliconFlowRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type siliconFlowResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// callLLMForUserProfile 调用SiliconFlow LLM生成用户画像
func callLLMForUserProfile(cfg *config.Config, prompt string) (string, string, error) {
	logger.Info("开始调用SiliconFlow LLM生成用户画像")
	segments := splitPrompt(cfg, prompt)
	return processSegmentsInParallel(cfg, segments)
}

// processSegmentsInParallel 并发处理多个提示词分段并合并结果
func processSegmentsInParallel(cfg *config.Config, segments []string) (string, string, error) {
	logger.Info("开始并发处理提示词分段", "segments_count", len(segments))

	// 并发处理各个分段
	var (
		segmentResults = make([]string, len(segments))
		errs           = make([]error, len(segments))
		wg             sync.WaitGroup
	)

	// 从配置获取最大并发数
	maxConcurrency := cfg.LLM.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 5 // 默认值
	}
	semaphore := make(chan struct{}, maxConcurrency)

	for idx, segment := range segments {
		wg.Add(1)
		go func(i int, segment string) {
			defer wg.Done()

			// 使用信号量限制并发数
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			logger.Info("并发处理提示词分段", "part", i+1, "total", len(segments))

			// 构建分段提示词，不需要指明是第几部分共几部分
			partPrompt := fmt.Sprintf("请分析以下用户数据中的关键词和兴趣，以JSON格式返回:\n\n%s", segment)

			// 直接调用API处理分段，避免递归调用
			profileJSON, _, err := callLLMDirectly(cfg, partPrompt)
			if err != nil {
				logger.Error("处理提示词分段失败", "part", i+1, "error", err)
				errs[i] = fmt.Errorf("处理提示词分段失败: %v", err)
				return
			}

			segmentResults[i] = profileJSON
		}(idx, segment)
	}
	wg.Wait()

	// 检查是否有错误
	for i, err := range errs {
		if err != nil {
			logger.Error("分段处理出错", "segment", i+1, "error", err)
			// 如果有错误，尝试继续处理其他成功的分段
			continue
		}
	}

	// 合并所有分段结果
	allKeywords := make(map[string]float64)
	var interests []string
	activityLevel := "low"
	userType := ""

	// 处理每个分段的结果
	for i, result := range segmentResults {
		if result == "" {
			continue // 跳过处理失败的分段
		}

		// 解析分段结果
		var profileData map[string]interface{}
		if err := json.Unmarshal([]byte(result), &profileData); err != nil {
			logger.Error("解析分段结果失败", "part", i+1, "error", err)
			continue
		}

		// 合并关键词
		if wk, ok := profileData["weighted_keywords"].([]interface{}); ok {
			for _, item := range wk {
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

		// 合并兴趣
		if ints, ok := profileData["interests"].([]interface{}); ok {
			for _, interest := range ints {
				if interestStr, ok := interest.(string); ok {
					interests = append(interests, interestStr)
				}
			}
		}

		// 更新活跃度和用户类型
		if al, ok := profileData["activity_level"].(string); ok && al == "high" {
			activityLevel = "high"
		} else if al == "medium" && activityLevel != "high" {
			activityLevel = "medium"
		}

		if ut, ok := profileData["user_type"].(string); ok && userType == "" {
			userType = ut
		}
	}

	// 构建最终结果
	var finalWeightedKeywords []models.WeightedKeyword
	for keyword, weight := range allKeywords {
		finalWeightedKeywords = append(finalWeightedKeywords, models.WeightedKeyword{
			Keyword: keyword,
			Weight:  weight,
		})
	}

	// 按权重排序
	sort.Slice(finalWeightedKeywords, func(i, j int) bool {
		return finalWeightedKeywords[i].Weight > finalWeightedKeywords[j].Weight
	})

	// 去重兴趣
	interests = utils.DeduplicateSlice(interests)

	// 提取关键词列表
	var keywords []string
	for _, wk := range finalWeightedKeywords {
		keywords = append(keywords, wk.Keyword)
	}

	// 将"无法确定"的用户类型统一为"新手"标签
	if userType == "" || userType == "无法确定" || userType == "新手" {
		userType = "新手"
	}

	// 如果interests为空，使用weighted_keywords的keyword合集
	if len(interests) == 0 && len(finalWeightedKeywords) > 0 {
		for _, wk := range finalWeightedKeywords {
			interests = append(interests, wk.Keyword)
		}
		logger.Info("使用关键词作为兴趣", "keywords_count", len(interests))
	}

	// 反之，如果有interests但没有weighted_keywords，今interests生成weighted_keywords
	if len(interests) > 0 && len(finalWeightedKeywords) == 0 {
		for i, interest := range interests {
			// 根据位置分配权重，首个兴趣权重最高
			weight := 0.9 - float64(i)*0.1
			if weight < 0.1 {
				weight = 0.1
			}
			finalWeightedKeywords = append(finalWeightedKeywords, models.WeightedKeyword{
				Keyword: interest,
				Weight:  weight,
			})
		}
		logger.Info("从兴趣生成加权关键词", "count", len(finalWeightedKeywords))
	}

	// 构建最终用户画像
	finalProfile := map[string]interface{}{
		"interests":         interests,
		"weighted_keywords": finalWeightedKeywords,
		"activity_level":    activityLevel,
		"user_type":         userType,
		"updated_at":        time.Now().Format(time.RFC3339),
	}

	// 如果没有兴趣和关键词，使用默认的"新手"画像
	if len(interests) == 0 && len(finalWeightedKeywords) == 0 {
		// 默认新手画像
		defaultInterests := []string{"区块链", "数字货币", "无链生态"}
		defaultKeywords := []models.WeightedKeyword{
			{Keyword: "区块链入门", Weight: 0.9},
			{Keyword: "数字货币基础", Weight: 0.85},
			{Keyword: "无链生态", Weight: 0.8},
			{Keyword: "DW20", Weight: 0.75},
			{Keyword: "钱包使用", Weight: 0.7},
		}

		finalProfile["interests"] = defaultInterests
		finalProfile["weighted_keywords"] = defaultKeywords
		finalProfile["user_type"] = "新手"

		// 更新关键词列表
		keywords = []string{}
		for _, wk := range defaultKeywords {
			keywords = append(keywords, wk.Keyword)
		}

		logger.Info("使用默认新手画像")
	}

	profileJSON, _ := json.Marshal(finalProfile)
	keywordsJSON, _ := json.Marshal(keywords)

	return string(profileJSON), string(keywordsJSON), nil
}

// callLLMDirectly 直接调用LLM API，避免递归调用
func callLLMDirectly(cfg *config.Config, prompt string) (string, string, error) {
	logger.Info("直接调用LLM API", "model", cfg.SiliconFlow.Model)

	// 记录提示词的前100个字符（避免日志过长）
	promptPreview := prompt
	if len(prompt) > 100 {
		promptPreview = prompt[:100] + "..."
	}
	logger.Info("LLM请求提示词预览", "prompt_preview", promptPreview)

	// 构建API请求
	apiKey := cfg.SiliconFlow.APIKey
	// 如果配置中的API Key是环境变量引用，则从环境变量中获取
	if strings.HasPrefix(apiKey, "${") && strings.HasSuffix(apiKey, "}") {
		envName := apiKey[2 : len(apiKey)-1]
		apiKey = os.Getenv(envName)
		logger.Info("从环境变量获取API Key", "env_var", envName)
	}

	reqBody := siliconFlowRequest{
		Model: cfg.SiliconFlow.Model,
		Messages: []message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		logger.Error("序列化请求体失败", "error", err)
		return "", "", err
	}

	logger.Info("LLM请求详情",
		"url", cfg.SiliconFlow.BaseURL+"/v1/chat/completions",
		"model", cfg.SiliconFlow.Model,
		"request_size", len(reqJSON))

	// 创建HTTP请求
	req, err := http.NewRequest("POST", cfg.SiliconFlow.BaseURL+"/v1/chat/completions", bytes.NewBuffer(reqJSON))
	if err != nil {
		logger.Error("创建HTTP请求失败", "error", err)
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 发送请求
	startTime := time.Now()
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	requestDuration := time.Since(startTime)

	logger.Info("LLM请求耗时", "duration_ms", requestDuration.Milliseconds())

	if err != nil {
		logger.Error("发送请求失败", "error", err, "duration_ms", requestDuration.Milliseconds())
		return "", "", err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取响应失败", "error", err)
		return "", "", err
	}

	// 记录响应状态和大小
	logger.Info("LLM响应状态", "status_code", resp.StatusCode, "response_size", len(body))

	if resp.StatusCode != http.StatusOK {
		// 记录错误响应内容
		responsePreview := string(body)
		if len(responsePreview) > 500 {
			responsePreview = responsePreview[:500] + "..."
		}
		logger.Error("API请求失败", "status", resp.StatusCode, "response", responsePreview)
		return "", "", fmt.Errorf("API请求失败: %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var sfResp siliconFlowResponse
	if err := json.Unmarshal(body, &sfResp); err != nil {
		logger.Error("解析响应失败", "error", err, "response_body_preview", string(body[:min(len(body), 200)]))
		return "", "", err
	}

	if len(sfResp.Choices) == 0 {
		logger.Error("API响应中没有内容", "response_body", string(body))
		return "", "", fmt.Errorf("API响应中没有内容")
	}

	// 提取LLM生成的内容
	content := sfResp.Choices[0].Message.Content
	logger.Info("成功获取LLM响应",
		"tokens_prompt", sfResp.Usage.PromptTokens,
		"tokens_completion", sfResp.Usage.CompletionTokens,
		"tokens_total", sfResp.Usage.TotalTokens,
		"finish_reason", sfResp.Choices[0].FinishReason)

	// 记录响应内容预览
	contentPreview := content
	if len(content) > 200 {
		contentPreview = content[:200] + "..."
	}
	logger.Info("LLM响应内容预览", "content_preview", contentPreview)

	// 解析LLM返回的JSON内容
	// 首先尝试从返回内容中提取JSON部分
	jsonContent := extractJSONFromText(content)
	logger.Info("提取的JSON内容", "json_content_preview", jsonContent[:min(len(jsonContent), 100)])

	var profileData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &profileData); err != nil {
		logger.Error("解析LLM返回的JSON内容失败", "error", err, "content", content)
		return "", "", err
	}

	logger.Info("成功解析LLM返回的JSON内容", "fields_count", len(profileData))

	// 提取带权重的关键词
	var weightedKeywords []models.WeightedKeyword
	var keywords []string

	if wk, ok := profileData["weighted_keywords"].([]interface{}); ok {
		for _, item := range wk {
			if kwObj, ok := item.(map[string]interface{}); ok {
				keyword, _ := kwObj["keyword"].(string)
				weight, _ := kwObj["weight"].(float64)
				if keyword != "" {
					weightedKeywords = append(weightedKeywords, models.WeightedKeyword{
						Keyword: keyword,
						Weight:  weight,
					})
					keywords = append(keywords, keyword)
				}
			}
		}
	}

	// 如果没有weighted_keywords但有interests，从 interests 生成 weighted_keywords
	if len(weightedKeywords) == 0 {
		if interests, ok := profileData["interests"].([]interface{}); ok {
			for i, interest := range interests {
				if interestStr, ok := interest.(string); ok && interestStr != "" {
					// 根据位置分配权重，首个兴趣权重最高
					weight := 0.9 - float64(i)*0.1
					if weight < 0.1 {
						weight = 0.1
					}
					weightedKeywords = append(weightedKeywords, models.WeightedKeyword{
						Keyword: interestStr,
						Weight:  weight,
					})
					keywords = append(keywords, interestStr)
				}
			}
			logger.Info("从兴趣生成加权关键词", "count", len(weightedKeywords))
		}
	}

	// 将带权重的关键词添加到响应中
	profileData["weighted_keywords"] = weightedKeywords
	profileData["updated_at"] = time.Now().Format(time.RFC3339)

	profileJSON, _ := json.Marshal(profileData)
	keywordsJSON, _ := json.Marshal(keywords)

	logger.Info("完成LLM处理",
		"keywords_count", len(keywords),
		"profile_json_size", len(profileJSON),
		"total_duration_ms", time.Since(startTime).Milliseconds())

	return string(profileJSON), string(keywordsJSON), nil
}

// extractJSONFromText 从文本中提取JSON部分
func extractJSONFromText(text string) string {
	// 查找文本中的JSON部分
	startIdx := strings.Index(text, "{")
	endIdx := strings.LastIndex(text, "}")

	if startIdx >= 0 && endIdx > startIdx {
		// 提取JSON部分
		jsonPart := text[startIdx : endIdx+1]
		logger.Info("成功从文本中提取JSON部分", "start_pos", startIdx, "end_pos", endIdx)
		return jsonPart
	}

	// 如果找不到JSON部分，尝试查找```json和```之间的内容
	startMarker := "```json"
	endMarker := "```"
	startIdx = strings.Index(text, startMarker)
	if startIdx >= 0 {
		startIdx += len(startMarker)
		endIdx = strings.Index(text[startIdx:], endMarker)
		if endIdx > 0 {
			jsonPart := text[startIdx : startIdx+endIdx]
			logger.Info("成功从代码块中提取JSON部分", "start_pos", startIdx, "end_pos", startIdx+endIdx)
			return strings.TrimSpace(jsonPart)
		}
	}

	// 如果仍然找不到，返回原始文本
	logger.Warn("无法从文本中提取JSON部分，返回原始文本")
	return text
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
