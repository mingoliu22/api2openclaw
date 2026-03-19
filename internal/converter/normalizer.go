package converter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ModelFamily 模型家族类型
type ModelFamily string

const (
	ModelFamilyDeepSeek ModelFamily = "deepseek"
	ModelFamilyQwen     ModelFamily = "qwen"
	ModelFamilyLlama    ModelFamily = "llama"
	ModelFamilyOpenAI   ModelFamily = "openai"
	ModelFamilyOther    ModelFamily = "other"
)

// Normalizer 格式归一化器
// 核心改造：将各模型差异格式 → 标准 OpenAI JSON
type Normalizer struct {
	config *ConverterConfig
}

// NewNormalizer 创建归一化器
func NewNormalizer(config *ConverterConfig) *Normalizer {
	return &Normalizer{config: config}
}

// Normalize 归一化响应为标准 OpenAI 格式
// 支持非流式完整响应转换
func (n *Normalizer) Normalize(raw []byte, modelFamily ModelFamily) (*openAIResponse, error) {
	// 检测输入格式并选择对应转换器
	switch modelFamily {
	case ModelFamilyDeepSeek:
		return n.normalizeDeepSeek(raw)
	case ModelFamilyQwen:
		return n.normalizeQwen(raw)
	case ModelFamilyOpenAI, ModelFamilyOther:
		return n.normalizeOpenAI(raw)
	default:
		// 尝试自动检测
		return n.autoDetectAndNormalize(raw)
	}
}

// NormalizeStreamChunk 归一化流式响应 chunk
func (n *Normalizer) NormalizeStreamChunk(raw []byte, modelFamily ModelFamily) (*openAIStreamChunk, error) {
	switch modelFamily {
	case ModelFamilyDeepSeek:
		return n.normalizeDeepSeekStreamChunk(raw)
	case ModelFamilyQwen:
		return n.normalizeQwenStreamChunk(raw)
	case ModelFamilyOpenAI, ModelFamilyOther:
		return n.normalizeOpenAIStreamChunk(raw)
	default:
		return n.autoDetectAndNormalizeStream(raw)
	}
}

// normalizeDeepSeek 归一化 DeepSeek 响应
func (n *Normalizer) normalizeDeepSeek(raw []byte) (*openAIResponse, error) {
	var deepseekResp struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Index        int `json:"index"`
			Message      struct {
				Role    string         `json:"role"`
				Content []interface{} `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(raw, &deepseekResp); err != nil {
		return nil, fmt.Errorf("unmarshal deepseek response: %w", err)
	}

	if len(deepseekResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in deepseek response")
	}

	srcChoice := deepseekResp.Choices[0]

	// 提取文本内容和处理多模态内容
	var textContent strings.Builder
	for _, item := range srcChoice.Message.Content {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if itemType, _ := itemMap["type"].(string); itemType == "text" {
				if text, ok := itemMap["text"].(string); ok {
					textContent.WriteString(text)
				}
			}
		}
	}

	// 构建标准 OpenAI 响应
	resp := &openAIResponse{
		ID:      deepseekResp.ID,
		Object:  "chat.completion",
		Created: deepseekResp.Created,
		Model:   deepseekResp.Model,
		Choices: []openAIChoice{
			{
				Index: srcChoice.Index,
				Message: OpenAIMessage{
					Role:    srcChoice.Message.Role,
					Content: textContent.String(),
				},
				FinishReason: mapFinishReason(srcChoice.FinishReason),
			},
		},
	}

	return resp, nil
}

// normalizeQwen 归一化 Qwen 响应
func (n *Normalizer) normalizeQwen(raw []byte) (*openAIResponse, error) {
	// Qwen (Ollama) 格式
	var qwenResp struct {
		Model     string `json:"model"`
		CreatedAt string `json:"created_at"`
		Message   struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Done bool `json:"done"`
		// 流式响应会有这些字段
		PromptEvalCount   *int `json:"prompt_eval_count,omitempty"`
		EvalCount         *int `json:"eval_count,omitempty"`
		// 非流式响应可能使用 choices 格式
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices,omitempty"`
	}

	if err := json.Unmarshal(raw, &qwenResp); err != nil {
		return nil, fmt.Errorf("unmarshal qwen response: %w", err)
	}

	// 检查是否为 choices 格式（非流式）
	if len(qwenResp.Choices) > 0 {
		srcChoice := qwenResp.Choices[0]
		totalTokens := 0
		if qwenResp.PromptEvalCount != nil && qwenResp.EvalCount != nil {
			totalTokens = *qwenResp.PromptEvalCount + *qwenResp.EvalCount
		}

		return &openAIResponse{
			ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   qwenResp.Model,
			Choices: []openAIChoice{
				{
					Index: 0,
					Message: OpenAIMessage{
						Role:    srcChoice.Message.Role,
						Content: srcChoice.Message.Content,
					},
					FinishReason: mapFinishReason(srcChoice.FinishReason),
				},
			},
			Usage: &openAIUsage{
				PromptTokens:     safeDeref(qwenResp.PromptEvalCount),
				CompletionTokens: safeDeref(qwenResp.EvalCount),
				TotalTokens:      totalTokens,
			},
		}, nil
	}

	// message 格式（可能是流式的单个 chunk）
	return &openAIResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   qwenResp.Model,
		Choices: []openAIChoice{
			{
				Index: 0,
				Message: OpenAIMessage{
					Role:    qwenResp.Message.Role,
					Content: qwenResp.Message.Content,
				},
				FinishReason: "stop",
			},
		},
	}, nil
}

// normalizeOpenAI 归一化 OpenAI 响应（直接透传）
func (n *Normalizer) normalizeOpenAI(raw []byte) (*openAIResponse, error) {
	var resp OpenAIResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal openai response: %w", err)
	}

	// 标准格式直接返回
	return &resp, nil
}

// normalizeDeepSeekStreamChunk 归一化 DeepSeek 流式 chunk
func (n *Normalizer) normalizeDeepSeekStreamChunk(raw []byte) (*openAIStreamChunk, error) {
	var deepseekChunk struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Index int `json:"index"`
			Delta struct {
				Role    string         `json:"role,omitempty"`
				Content []interface{} `json:"content,omitempty"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason,omitempty"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(raw, &deepseekChunk); err != nil {
		return nil, fmt.Errorf("unmarshal deepseek stream chunk: %w", err)
	}

	if len(deepseekChunk.Choices) == 0 {
		return &openAIStreamChunk{}, nil
	}

	srcChoice := deepseekChunk.Choices[0]

	// 提取文本内容
	var contentText string
	for _, item := range srcChoice.Delta.Content {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if itemType, _ := itemMap["type"].(string); itemType == "text" {
				if text, ok := itemMap["text"].(string); ok {
					contentText += text
				}
			}
		}
	}

	return &openAIStreamChunk{
		ID:      deepseekChunk.ID,
		Object:  "chat.completion.chunk",
		Created: deepseekChunk.Created,
		Model:   deepseekChunk.Model,
		Choices: []openAIStreamChoice{
			{
				Index: srcChoice.Index,
				Delta: openAIStreamDelta{
					Role:    optionalString(srcChoice.Delta.Role),
					Content: optionalString(contentText),
				},
				FinishReason: optionalString(srcChoice.FinishReason),
			},
		},
	}, nil
}

// normalizeQwenStreamChunk 归一化 Qwen 流式 chunk
func (n *Normalizer) normalizeQwenStreamChunk(raw []byte) (*openAIStreamChunk, error) {
	var qwenChunk struct {
		Model     string `json:"model"`
		CreatedAt string `json:"created_at"`
		Message   struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Done              bool  `json:"done"`
		PromptEvalCount   *int  `json:"prompt_eval_count,omitempty"`
		EvalCount         *int  `json:"eval_count,omitempty"`
		Context           []int `json:"context,omitempty"`
	}

	if err := json.Unmarshal(raw, &qwenChunk); err != nil {
		return nil, fmt.Errorf("unmarshal qwen stream chunk: %w", err)
	}

	chunk := &openAIStreamChunk{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   qwenChunk.Model,
		Choices: []openAIStreamChoice{
			{
				Index: 0,
				Delta: openAIStreamDelta{
					Role:    optionalString(qwenChunk.Message.Role),
					Content: qwenChunk.Message.Content,
				},
			},
		},
	}

	if qwenChunk.Done {
		chunk.Choices[0].FinishReason = strPtr("stop")
	}

	return chunk, nil
}

// normalizeOpenAIStreamChunk 归一化 OpenAI 流式 chunk（直接透传）
func (n *Normalizer) normalizeOpenAIStreamChunk(raw []byte) (*openAIStreamChunk, error) {
	var chunk OpenAIStreamChunk
	if err := json.Unmarshal(raw, &chunk); err != nil {
		return nil, fmt.Errorf("unmarshal openai stream chunk: %w", err)
	}
	return &chunk, nil
}

// autoDetectAndNormalize 自动检测格式并归一化
func (n *Normalizer) autoDetectAndNormalize(raw []byte) (*openAIResponse, error) {
	var rawMap map[string]interface{}
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// 检测特征字段
	hasContentArray := false
	hasMessageContent := false

	if choices, ok := rawMap["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].([]interface{}); ok {
					hasContentArray = true
				}
				if _, hasContentKey := message["content"]; hasContentKey {
					hasMessageContent = true
				}
			}
		}
	}

	// 根据特征判断格式
	if hasContentArray {
		// DeepSeek 风格（content 是数组）
		return n.normalizeDeepSeek(raw)
	}

	// 默认按 OpenAI 格式处理
	return n.normalizeOpenAI(raw)
}

// autoDetectAndNormalizeStream 自动检测流式格式并归一化
func (n *Normalizer) autoDetectAndNormalizeStream(raw []byte) (*openAIStreamChunk, error) {
	var rawMap map[string]interface{}
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return nil, fmt.Errorf("unmarshal stream chunk: %w", err)
	}

	// 检测 Ollama/Qwen 特征
	if _, hasDone := rawMap["done"]; hasDone {
		if _, hasMessage := rawMap["message"]; hasMessage {
			return n.normalizeQwenStreamChunk(raw)
		}
	}

	// 默认按 OpenAI 格式处理
	return n.normalizeOpenAIStreamChunk(raw)
}

// === 标准响应结构 ===

// openAIResponse 标准 OpenAI 响应
type openAIResponse struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   *OpenAIUsage  `json:"usage,omitempty"`
}

// openAIChoice OpenAI 选择
type openAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// openAIUsage OpenAI 使用量
type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// openAIStreamChunk OpenAI 流式 chunk
type openAIStreamChunk struct {
	ID      string                `json:"id"`
	Object  string                `json:"object"`
	Created int64                 `json:"created"`
	Model   string                `json:"model"`
	Choices []OpenAIStreamChoice  `json:"choices"`
}

// openAIStreamChoice OpenAI 流式选择
type openAIStreamChoice struct {
	Index        int                `json:"index"`
	Delta        OpenAIStreamDelta  `json:"delta"`
	FinishReason *string            `json:"finish_reason,omitempty"`
}

// openAIStreamDelta OpenAI 流式增量
type openAIStreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// === 辅助函数 ===

// mapFinishReason 映射 finish_reason
func mapFinishReason(reason string) string {
	switch reason {
	case "tool_calls", "tool_use":
		return "tool_calls"
	case "length", "max_tokens":
		return "length"
	case "stop", "":
		return "stop"
	default:
		return reason
	}
}

// optionalString 返回字符串指针，空字符串返回 nil
func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// strPtr 返回字符串指针
func strPtr(s string) *string {
	return &s
}

// safeDeref 安全解引用，nil 返回 0
func safeDeref(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
