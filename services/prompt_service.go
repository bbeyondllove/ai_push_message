package services

import (
	"ai_push_message/config"
	"ai_push_message/repository"
	"ai_push_message/utils"
	"fmt"
	"strings"
)

// 不再需要simplifyPrompt函数，直接使用splitPrompt进行分段处理

// splitPrompt 将提示词分成多个部分，参考ai_center中的群总结分段逻辑
func splitPrompt(cfg *config.Config, prompt string) []string {
	lines := strings.Split(prompt, "\n")

	// 找到内容部分的开始和结束
	contentStart, contentEnd := -1, -1
	for i, line := range lines {
		if strings.Contains(line, "社区发帖内容") {
			contentStart = i + 1
		} else if contentStart > 0 && strings.Contains(line, "群聊消息内容") {
			contentEnd = i - 1
			break
		}
	}

	// 如果没有找到内容部分，返回原始提示词
	if contentStart < 0 || contentEnd < contentStart {
		return []string{prompt}
	}

	// 提取前导部分和后续部分
	prefix := strings.Join(lines[:contentStart], "\n")
	suffix := strings.Join(lines[contentEnd+1:], "\n")

	// 设置最大token数，从配置文件中读取
	maxTokens := cfg.SiliconFlow.MaxTokenLength / 2 // 分段时使用一半的最大token长度，确保安全

	// 使用utils包中的SplitTextByTokens函数分段
	contentText := strings.Join(lines[contentStart:contentEnd+1], "\n")
	contentBlocks := utils.SplitTextByTokens(contentText, maxTokens)

	// 组合成完整的提示词块
	var promptBlocks []string
	for _, block := range contentBlocks {
		promptBlocks = append(promptBlocks, prefix+"\n"+block+"\n"+suffix)
	}

	return promptBlocks
}

// buildUserAnalysisPrompt 构建用户分析提示词
func buildUserAnalysisPrompt(cid string, userData *repository.CombinedUserData) string {
	prompt := fmt.Sprintf(`请分析用户 %s 的行为数据，生成用户画像标签。

用户数据来源：
- 社区发帖数据：%d 条
- 群聊消息数据：%d 条  
- 活跃群组：%v
- 群组兴趣：%v

社区发帖内容：
%s

群聊消息内容：
%s

请基于以上数据分析用户的兴趣偏好，生成能够搜索"DW20与比特币的比较"和"无链常见问题问答"等知识库内容的标签。

要求：
1. 提取用户关注的核心话题和兴趣点
2. 识别用户在区块链、数字货币、无链生态等方面的参与度
3. 生成便于知识库搜索的关键词标签，并为每个关键词分配权重（0-1之间的浮点数）
4. 标签应涵盖：技术兴趣、投资偏好、产品使用、问题类型等维度
5. 关键词按权重从高到低排序，权重高的关键词表示用户更关注的内容

请以JSON格式返回分析结果：
{
  "interests": ["兴趣1", "兴趣2"],
  "weighted_keywords": [
    {"keyword": "关键词1", "weight": 0.95},
    {"keyword": "关键词2", "weight": 0.85},
    {"keyword": "关键词3", "weight": 0.75}
  ],
  "activity_level": "high/medium/low",
  "user_type": "投资者/技术爱好者/新手"
}`,
		cid,
		len(userData.CommunityPosts),
		len(userData.GroupMessages),
		userData.ActiveGroups,
		userData.GroupInterests,
		strings.Join(userData.CommunityPosts, "\n---\n"),
		strings.Join(userData.GroupMessages, "\n---\n"))

	return prompt
}
