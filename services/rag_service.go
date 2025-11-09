package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ai_push_message/config"
	"ai_push_message/logger"
	"ai_push_message/models"
	"ai_push_message/utils"
)

type ragResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Query   string `json:"query"`
		Results []struct {
			ChunkID     string  `json:"chunk_id"`
			DocumentID  string  `json:"document_id"`
			KnowledgeID string  `json:"knowledge_id"`
			Title       string  `json:"title"`
			Content     string  `json:"content"`
			Summary     string  `json:"summary"`
			Score       float64 `json:"score"`
		} `json:"results"`
		Total int `json:"total"`
	} `json:"data"`
}

func CallRAG(cfg *config.Config, query string) ([]models.RecommendationItem, error) {
	logger.Info("调用RAG服务搜索关键词", "query", query)

	payload := map[string]any{
		"knowledge_ids": cfg.RAG.KnowledgeIDs,
		"query":         query,
		"threshold":     cfg.RAG.Threshold, // 使用rag.threshold配置
		"top_k":         cfg.RAG.TopK,      // 使用rag.topk配置
	}
	b, _ := json.Marshal(payload)

	logger.Info("RAG请求参数", "payload", string(b))
	logger.Info("RAG服务URL", "url", cfg.RAG.URL)

	req, err := http.NewRequest("POST", cfg.RAG.URL, bytes.NewReader(b))
	if err != nil {
		logger.Error("创建RAG请求失败", "error", err)
		return nil, err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.RAG.APIKey))
	req.Header.Set("Content-Type", "application/json")

	// 使用配置的超时时间
	timeout := time.Duration(cfg.RAG.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second // 默认超时
	}
	client := &http.Client{Timeout: timeout}

	logger.Info("发送RAG请求...")
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("RAG请求失败", "error", err)
		return nil, fmt.Errorf("RAG服务连接失败: %v", err)
	}
	defer resp.Body.Close()

	logger.Info("RAG响应状态码", "status_code", resp.StatusCode)

	// 读取响应体
	var bodyBytes []byte
	bodyBytes, _ = io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	logger.Info("RAG响应内容", "response", string(bodyBytes))

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		logger.Error("RAG服务返回错误状态码", "status_code", resp.StatusCode, "response", string(bodyBytes))
		return nil, fmt.Errorf("RAG服务错误 (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var rr ragResp
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&rr); err != nil {
		logger.Error("解析RAG响应失败", "error", err)
		return nil, fmt.Errorf("解析RAG响应失败: %v", err)
	}

	// 检查业务状态码
	if rr.Code != 0 {
		logger.Error("RAG服务业务错误", "code", rr.Code, "message", rr.Message)
		return nil, fmt.Errorf("RAG服务业务错误: %s (错误码: %d)", rr.Message, rr.Code)
	}

	logger.Info("RAG响应解析结果", "code", rr.Code, "message", rr.Message, "result_count", len(rr.Data.Results))

	// 创建RAG内容格式化器，根据配置决定是否启用口语化处理
	var formatter *utils.RAGContentFormatter
	if cfg.SiliconFlow.APIKey != "" && cfg.SiliconFlow.Model != "" {
		// 如果配置了SiliconFlow，启用口语化处理
		siliconFlowConfig := &utils.SiliconFlowConfig{
			APIKey:  cfg.SiliconFlow.APIKey,
			Model:   cfg.SiliconFlow.Model,
			BaseURL: cfg.SiliconFlow.BaseURL,
		}
		formatter = utils.NewRAGContentFormatterWithColloquialization(siliconFlowConfig)
	} else {
		// 否则只使用基本格式化功能
		formatter = utils.NewRAGContentFormatter()
	}

	items := make([]models.RecommendationItem, 0, len(rr.Data.Results))
	for _, r := range rr.Data.Results {
		// 分别处理标题和内容，而不是一起处理
		formattedTitle := formatter.RemoveMarkdownHeaders(r.Title)
		formattedContent := formatter.RemoveMarkdownHeaders(r.Content)

		// 添加对空内容和重复内容的检查
		if formattedTitle == "" || formattedContent == "" {
			logger.Info("跳过空的推荐项", "document_id", r.DocumentID)
			continue
		}

		// 如果启用了口语化处理，分别对标题和内容进行口语化处理
		if formatter.EnableColloquialization && formatter.SiliconFlowConfig != nil {
			colContent, err := formatter.ColloquializeContent(formattedContent)
			if err != nil {
				logger.Error("内容口语化处理失败，使用原始格式化内容", "error", err)
			} else {
				formattedContent = colContent
			}
		}

		items = append(items, models.RecommendationItem{
			Source:  "rag",
			Title:   formattedTitle,
			Content: formattedContent,
			URL:     "",
			Score:   r.Score,
			RefID:   r.DocumentID,
		})
	}

	logger.Info("生成的推荐项数量", "count", len(items))
	return items, nil
}
