package converter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// DeepSeekParser DeepSeek 格式解析器
type DeepSeekParser struct {
	config *ConverterConfig
}

// DeepSeekResponse DeepSeek 响应结构
type DeepSeekResponse struct {
	Choices []DeepSeekChoice `json:"choices"`
	Usage   DeepSeekUsage    `json:"usage"`
}

// DeepSeekChoice 消息选择
type DeepSeekChoice struct {
	Message    DeepSeekMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// DeepSeekMessage 消息内容
type DeepSeekMessage struct {
	Role    string               `json:"role"`
	Content []DeepSeekContentItem `json:"content"`
}

// DeepSeekContentItem 内容项
type DeepSeekContentItem struct {
	Type       string                 `json:"type"`
	Text       string                 `json:"text,omitempty"`
	ID         string                 `json:"id,omitempty"`         // for tool_use
	Name       string                 `json:"name,omitempty"`       // for tool_use
	Input      map[string]interface{} `json:"input,omitempty"`      // for tool_use
	ToolUseID  string                 `json:"tool_use_id,omitempty"` // for tool_result
	Content    interface{}            `json:"content,omitempty"`    // for tool_result
	IsError    bool                   `json:"is_error,omitempty"`   // for tool_result
}

// DeepSeekUsage 使用量
type DeepSeekUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewDeepSeekParser 创建 DeepSeek 解析器
func NewDeepSeekParser(config *ConverterConfig) *DeepSeekParser {
	return &DeepSeekParser{config: config}
}

// Parse 解析 DeepSeek 响应
func (p *DeepSeekParser) Parse(data []byte) (*DeepSeekResponse, error) {
	var resp DeepSeekResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal deepseek response: %w", err)
	}
	return &resp, nil
}

// ExtractText 提取文本内容
func (p *DeepSeekParser) ExtractText(resp *DeepSeekResponse) (string, error) {
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	content := resp.Choices[0].Message.Content
	var result string

	for _, item := range content {
		if item.Type == "text" {
			result += item.Text
		}
	}

	return result, nil
}

// ConvertToOpenClaw 转换为 OpenClaw 纯文本格式
func (p *DeepSeekParser) ConvertToOpenClaw(data []byte) ([]byte, error) {
	resp, err := p.Parse(data)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	// 提取文本
	text, err := p.ExtractText(resp)
	if err != nil {
		return nil, err
	}

	// 使用模板格式化
	template := p.config.Templates.Message
	output := fmt.Sprintf(template, text)

	// 如果需要，附加使用量信息
	if p.config.IncludeUsage {
		usage := fmt.Sprintf("\n\nusage: prompt=%d completion=%d total=%d",
			resp.Usage.PromptTokens,
			resp.Usage.CompletionTokens,
			resp.Usage.TotalTokens)
		output += usage
	}

	return []byte(output), nil
}

// ConvertStream 转换流式响应
func (p *DeepSeekParser) ConvertStream(r io.Reader, w io.Writer) error {
	decoder := json.NewDecoder(r)

	for {
		var chunk map[string]interface{}
		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decode stream chunk: %w", err)
		}

		// 提取 choices[0].delta.content
		if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if delta, ok := choice["delta"].(map[string]interface{}); ok {
					if content, ok := delta["content"].([]interface{}); ok {
						// 提取文本内容
						for _, item := range content {
							if itemMap, ok := item.(map[string]interface{}); ok {
								if itemType, ok := itemMap["type"].(string); ok && itemType == "text" {
									if text, ok := itemMap["text"].(string); ok {
										fmt.Fprintf(w, p.config.Templates.StreamChunk, text)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// HasToolCalls 检查响应是否包含工具调用
func (p *DeepSeekParser) HasToolCalls(data []byte) (bool, error) {
	resp, err := p.Parse(data)
	if err != nil {
		return false, err
	}

	if len(resp.Choices) == 0 {
		return false, nil
	}

	for _, item := range resp.Choices[0].Message.Content {
		if item.Type == "tool_use" {
			return true, nil
		}
		if item.Type == "tool_result" {
			return true, nil
		}
	}

	return false, nil
}

// ExtractToolCalls 提取工具调用
func (p *DeepSeekParser) ExtractToolCalls(data []byte) ([]ToolCall, error) {
	resp, err := p.Parse(data)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	var toolCalls []ToolCall

	for _, item := range resp.Choices[0].Message.Content {
		if item.Type == "tool_use" {
			// 序列化参数
			var arguments json.RawMessage
			if item.Input != nil {
				arguments, _ = json.Marshal(item.Input)
			}

			toolCalls = append(toolCalls, ToolCall{
				ID:   item.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      item.Name,
					Arguments: arguments,
				},
			})
		}
	}

	return toolCalls, nil
}

// ConvertToolCallsToOpenAI 将 DeepSeek 工具调用转换为 OpenAI 格式
func (p *DeepSeekParser) ConvertToolCallsToOpenAI(data []byte) ([]byte, error) {
	resp, err := p.Parse(data)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	message := resp.Choices[0].Message
	var textContent strings.Builder
	var toolCalls []ToolCall

	for _, item := range message.Content {
		switch item.Type {
		case "text":
			textContent.WriteString(item.Text)
		case "tool_use":
			var arguments json.RawMessage
			if item.Input != nil {
				arguments, _ = json.Marshal(item.Input)
			}

			toolCalls = append(toolCalls, ToolCall{
				ID:   item.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      item.Name,
					Arguments: arguments,
				},
			})
		}
	}

	// 构建 OpenAI 格式响应
	openAIResp := map[string]interface{}{
		"id":      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "deepseek-chat",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":       "assistant",
					"content":    textContent.String(),
					"tool_calls": toolCalls,
				},
				"finish_reason": func() string {
					if len(toolCalls) > 0 {
						return "tool_calls"
					}
					return "stop"
				}(),
			},
		},
	}

	return json.Marshal(openAIResp)
}
