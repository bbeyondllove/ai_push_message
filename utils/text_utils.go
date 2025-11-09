package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DeduplicateSlice 去重字符串切片
func DeduplicateSlice(input []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, val := range input {
		val = strings.TrimSpace(val)
		if val != "" && !seen[val] {
			result = append(result, val)
			seen[val] = true
		}
	}

	return result
}

// CalculateTokens 计算文本的token数量：中文字符2token，英文单词1token
func CalculateTokens(text string) int {
	chinese := 0
	english := 0

	// 计算中文字符数
	for _, r := range []rune(text) {
		if r >= '\u4e00' && r <= '\u9fa5' {
			chinese++
		}
	}

	// 计算英文单词数
	english = len(strings.FieldsFunc(text, func(r rune) bool {
		return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z')
	}))

	return chinese*2 + english
}

// Min 返回两个整数中的较小值
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// IndexOf 返回元素在切片中的索引，如果不存在则返回-1
func IndexOf(slice []string, element string) int {
	for i, e := range slice {
		if e == element {
			return i
		}
	}
	return -1
}

// SplitTextByTokens 将文本按token数量分段
func SplitTextByTokens(text string, maxTokens int) []string {
	lines := strings.Split(text, "\n")
	var segments []string

	var currentSegment []string
	tokenCount := 0

	for _, line := range lines {
		lineTokens := CalculateTokens(line)

		// 如果当前段加上这一行会超过最大token数，则开始新的段
		if tokenCount+lineTokens > maxTokens && len(currentSegment) > 0 {
			segments = append(segments, strings.Join(currentSegment, "\n"))
			currentSegment = []string{}
			tokenCount = 0
		}

		// 添加当前行到当前段
		currentSegment = append(currentSegment, line)
		tokenCount += lineTokens
	}

	// 添加最后一个段
	if len(currentSegment) > 0 {
		segments = append(segments, strings.Join(currentSegment, "\n"))
	}

	return segments
}

// FilterSpecialSymbols 过滤文本中的特殊符号，只保留常见标点符号和正常内容
func FilterSpecialSymbols(text string) string {
	// 定义要保留的常见标点符号
	commonPunctuation := map[rune]bool{
		'，': true, '。': true, '！': true, '？': true, '：': true, '；': true,
		'、': true, '（': true, '）': true,
		'【': true, '】': true, '《': true, '》': true, '—': true,
		',': true, '.': true, '!': true, '?': true, ':': true, ';': true,
		'"': true, '\'': true, '(': true, ')': true, '[': true, ']': true,
		'{': true, '}': true, '<': true, '>': true, '-': true, '_': true,
		'+': true, '=': true, '/': true, '\\': true, '|': true, ' ': true,
		'\n': true, '\r': true, '\t': true,
	}

	var result strings.Builder
	for _, r := range []rune(text) {
		// 保留中文字符、英文字母、数字和常见标点符号
		if (r >= '\u4e00' && r <= '\u9fa5') || // 中文字符
			(r >= 'A' && r <= 'Z') || // 大写英文字母
			(r >= 'a' && r <= 'z') || // 小写英文字母
			(r >= '0' && r <= '9') || // 数字
			commonPunctuation[r] { // 常见标点符号
			result.WriteRune(r)
		}
	}

	return result.String()
}

// RAGContentFormatter RAG内容格式化工具
type RAGContentFormatter struct {
	// 是否启用SiliconFlow口语化处理
	EnableColloquialization bool
	// SiliconFlow配置（如果启用口语化处理）
	SiliconFlowConfig *SiliconFlowConfig
}

// SiliconFlowConfig SiliconFlow配置
type SiliconFlowConfig struct {
	APIKey  string
	Model   string
	BaseURL string
}

// SiliconFlowRequest SiliconFlow API请求结构
type SiliconFlowRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// SiliconFlowResponse SiliconFlow API响应结构
type SiliconFlowResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// NewRAGContentFormatter 创建RAG内容格式化器
func NewRAGContentFormatter() *RAGContentFormatter {
	return &RAGContentFormatter{
		EnableColloquialization: false, // 默认不启用口语化处理
	}
}

// NewRAGContentFormatterWithColloquialization 创建带口语化功能的RAG内容格式化器
func NewRAGContentFormatterWithColloquialization(config *SiliconFlowConfig) *RAGContentFormatter {
	return &RAGContentFormatter{
		EnableColloquialization: true,
		SiliconFlowConfig:       config,
	}
}

// FormatTitleAndContent 分别格式化标题和内容
// 主要功能：分别对标题和内容进行移除markdown标题格式和口语化处理
func (f *RAGContentFormatter) FormatTitleAndContent(title, content string) (string, string) {
	if title == "" && content == "" {
		return "", ""
	}

	// 分别移除markdown标题格式
	formattedTitle := f.RemoveMarkdownHeaders(title)
	formattedContent := f.RemoveMarkdownHeaders(content)

	return strings.TrimSpace(formattedTitle), strings.TrimSpace(formattedContent)
}

// RemoveMarkdownHeaders 移除markdown标题格式
func (f *RAGContentFormatter) RemoveMarkdownHeaders(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		// 移除##开头的标题，保留内容
		if strings.HasPrefix(strings.TrimSpace(line), "##") {
			// 提取标题内容，作为普通段落
			title := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "##"))
			if title != "" {
				result = append(result, title)
			}
		} else if strings.TrimSpace(line) != "" {
			// 处理可能存在的"标题："和"内容："前缀
			trimmedLine := strings.TrimSpace(line)
			if strings.HasPrefix(trimmedLine, "标题：") {
				// 移除"标题："前缀
				titleContent := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "标题："))
				if titleContent != "" {
					result = append(result, titleContent)
				}
			} else if strings.HasPrefix(trimmedLine, "内容：") {
				// 移除"内容："前缀
				contentContent := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "内容："))
				if contentContent != "" {
					result = append(result, contentContent)
				}
			} else {
				// 保留非空行
				result = append(result, line)
			}
		}
	}

	return strings.Join(result, "\n")
}

// ColloquializeContent 对单个内容进行口语化处理
func (f *RAGContentFormatter) ColloquializeContent(content string) (string, error) {
	if !f.EnableColloquialization || f.SiliconFlowConfig == nil {
		return content, nil
	}

	// 构建口语化处理的提示词
	prompt := fmt.Sprintf(`请对以下内容进行口语化处理，要求：

**要求：**
- 使用口语化、易懂的表述方式，避免过于学术化的术语和复杂表述
- 用简单直接的语言解释技术概念
- 优先使用"可以"、"能够"、"帮助"等日常词汇
- 保持内容的完整性和准确性,内容必须详细深入
- 避免使用语气助词，如"啊"、"呀"、"呢"等

原始内容：
%s

请直接返回口语化处理后的内容，不要添加任何其他说明或文本。`, content)

	// 调用SiliconFlow API
	result, err := f.callSiliconFlowAPI(prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(result), nil
}

// extractJSONFromString 从字符串中提取JSON内容
func (f *RAGContentFormatter) extractJSONFromString(input string) string {
	// 如果输入为空，直接返回
	if input == "" {
		return input
	}

	// 查找第一个 '{' 和最后一个 '}' 之间的内容
	start := strings.Index(input, "{")
	end := strings.LastIndex(input, "}")

	// 如果没有找到有效的JSON结构，尝试清理整个字符串
	if start == -1 || end == -1 || start >= end {
		// 清理常见的干扰字符
		cleaned := strings.TrimSpace(input)
		cleaned = strings.ReplaceAll(cleaned, "```json", "")
		cleaned = strings.ReplaceAll(cleaned, "```", "")
		cleaned = strings.TrimPrefix(cleaned, "json")
		cleaned = strings.TrimSpace(cleaned)
		return cleaned
	}

	// 提取JSON部分
	jsonStr := input[start : end+1]

	// 清理可能的非ASCII字符和其他干扰字符
	jsonStr = strings.ReplaceAll(jsonStr, "```json", "")
	jsonStr = strings.ReplaceAll(jsonStr, "```", "")
	jsonStr = strings.TrimPrefix(jsonStr, "json")
	jsonStr = strings.TrimSpace(jsonStr)

	// 进一步清理可能存在的特殊字符
	jsonStr = strings.Map(func(r rune) rune {
		if r >= 32 && r <= 126 || r == '\n' || r == '\r' || r == '\t' {
			return r
		}
		// 移除非ASCII字符
		return -1
	}, jsonStr)

	return jsonStr
}

// callSiliconFlowAPI 调用SiliconFlow API
func (f *RAGContentFormatter) callSiliconFlowAPI(prompt string) (string, error) {
	// 获取API Key
	apiKey := f.SiliconFlowConfig.APIKey
	if strings.HasPrefix(apiKey, "${") && strings.HasSuffix(apiKey, "}") {
		// 从环境变量获取API Key
		envName := apiKey[2 : len(apiKey)-1]
		apiKey = os.Getenv(envName)
		if apiKey == "" {
			return "", fmt.Errorf("环境变量 %s 未设置", envName)
		}
	}

	// 构建请求体
	reqBody := SiliconFlowRequest{
		Model: f.SiliconFlowConfig.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// 序列化请求体
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %v", err)
	}

	// 创建HTTP请求
	url := f.SiliconFlowConfig.BaseURL + "/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API请求失败: %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var sfResp SiliconFlowResponse
	if err := json.Unmarshal(body, &sfResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查响应内容
	if len(sfResp.Choices) == 0 {
		return "", fmt.Errorf("API响应中没有内容")
	}

	// 返回口语化后的内容
	colloquializedContent := sfResp.Choices[0].Message.Content
	return strings.TrimSpace(colloquializedContent), nil
}

// SetColloquializationConfig 设置口语化配置
func (f *RAGContentFormatter) SetColloquializationConfig(config *SiliconFlowConfig) {
	f.EnableColloquialization = true
	f.SiliconFlowConfig = config
}

// DisableColloquialization 禁用口语化处理
func (f *RAGContentFormatter) DisableColloquialization() {
	f.EnableColloquialization = false
	f.SiliconFlowConfig = nil
}
