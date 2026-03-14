package converter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ToolCall 工具调用标准格式
type ToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`     // "function"
	Function FunctionCall    `json:"function"`
}

// FunctionCall 函数调用
type FunctionCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolCallResult 工具调用结果
type ToolCallResult struct {
	ToolCallID string `json:"tool_call_id"`
	Role       string `json:"role"`
	Content    string `json:"content"`
}

// --- OpenAI 格式 ---

// OpenAIMessage OpenAI 消息格式
type OpenAIMessage struct {
	Role       string       `json:"role"`
	Content    string       `json:"content,omitempty"`
	ToolCalls  []ToolCall   `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"` // 用于工具结果
}

// OpenAIResponse OpenAI 响应格式
type OpenAIResponse struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Created int64           `json:"created"`
	Model   string          `json:"model"`
	Choices []OpenAIChoice  `json:"choices"`
}

// OpenAIChoice OpenAI 选择
type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// --- DeepSeek 格式 ---

// DeepSeekToolUse DeepSeek 工具使用格式
type DeepSeekToolUse struct {
	Type       string                 `json:"type"`       // "tool_use"
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"input,omitempty"` // DeepSeek uses "input"
}

// --- Anthropic Claude 格式 ---

// AnthropicToolUse Anthropic 工具使用格式
type AnthropicToolUse struct {
	Type       string                 `json:"type"` // "tool_use"
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Input      map[string]interface{} `json:"input"`
}

// AnthropicContentBlock Anthropic 内容块
type AnthropicContentBlock struct {
	Type       string                 `json:"type"` // "text", "tool_use", "tool_result"
	Text       string                 `json:"text,omitempty"`
	ID         string                 `json:"id,omitempty"`
	Name       string                 `json:"name,omitempty"`
	ToolUseID  string                 `json:"tool_use_id,omitempty"` // for tool_result
	Input      map[string]interface{} `json:"input,omitempty"`       // for tool_use
	Content    interface{}            `json:"content,omitempty"`     // for tool_result
	IsError    bool                   `json:"is_error,omitempty"`
}

// --- 转换器 ---

// ToolCallConverter 工具调用转换器
type ToolCallConverter struct {
	targetFormat string // "openai", "openclaw", "anthropic"
}

// NewToolCallConverter 创建工具调用转换器
func NewToolCallConverter(targetFormat string) *ToolCallConverter {
	return &ToolCallConverter{
		targetFormat: targetFormat,
	}
}

// ConvertFromOpenAI 从 OpenAI 格式转换
func (c *ToolCallConverter) ConvertFromOpenAI(data []byte) ([]byte, error) {
	var resp OpenAIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse openai response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return data, nil // 没有工具调用，返回原数据
	}

	message := resp.Choices[0].Message
	if len(message.ToolCalls) == 0 {
		return data, nil // 没有工具调用
	}

	switch c.targetFormat {
	case "openclaw":
		return c.convertToOpenClaw(&resp)
	case "anthropic":
		return c.convertToAnthropic(&resp)
	default:
		return data, nil
	}
}

// ConvertFromDeepSeek 从 DeepSeek 格式转换
func (c *ToolCallConverter) ConvertFromDeepSeek(data []byte) ([]byte, error) {
	var deepseekResp struct {
		Choices []struct {
			Message struct {
				Role    string         `json:"role"`
				Content []interface{} `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(data, &deepseekResp); err != nil {
		return nil, fmt.Errorf("parse deepseek response: %w", err)
	}

	if len(deepseekResp.Choices) == 0 {
		return data, nil
	}

	// 提取工具调用
	var toolCalls []ToolCall
	var textContent strings.Builder

	for _, item := range deepseekResp.Choices[0].Message.Content {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

	 itemType, _ := itemMap["type"].(string)

		switch itemType {
		case "text":
			if text, ok := itemMap["text"].(string); ok {
				textContent.WriteString(text)
			}
		case "tool_use":
			// DeepSeek 工具使用格式
			id, _ := itemMap["id"].(string)
			name, _ := itemMap["name"].(string)

			var parameters map[string]interface{}
			if input, ok := itemMap["input"].(map[string]interface{}); ok {
				parameters = input
			}

			arguments, _ := json.Marshal(parameters)

			toolCalls = append(toolCalls, ToolCall{
				ID:   id,
				Type: "function",
				Function: FunctionCall{
					Name:      name,
					Arguments: json.RawMessage(arguments),
				},
			})
		}
	}

	// 构建标准 OpenAI 格式响应
	openaiResp := OpenAIResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Model:   "deepseek-chat",
		Choices: []OpenAIChoice{
			{
				Index: 0,
				Message: OpenAIMessage{
					Role:       "assistant",
					Content:    textContent.String(),
					ToolCalls:  toolCalls,
				},
				FinishReason: "tool_calls",
			},
		},
	}

	return json.Marshal(openaiResp)
}

// ConvertFromAnthropic 从 Anthropic 格式转换
func (c *ToolCallConverter) ConvertFromAnthropic(data []byte) ([]byte, error) {
	var anthropicResp struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		Role         string `json:"role"`
		Content      []AnthropicContentBlock `json:"content"`
		Model        string `json:"model"`
		StopReason   string `json:"stop_reason"`
	}

	if err := json.Unmarshal(data, &anthropicResp); err != nil {
		return nil, fmt.Errorf("parse anthropic response: %w", err)
	}

	var toolCalls []ToolCall
	var textContent strings.Builder

	for _, block := range anthropicResp.Content {
		switch block.Type {
		case "text":
			textContent.WriteString(block.Text)
		case "tool_use":
			arguments, _ := json.Marshal(block.Input)

			toolCalls = append(toolCalls, ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      block.Name,
					Arguments: json.RawMessage(arguments),
				},
			})
		}
	}

	// 构建标准 OpenAI 格式响应
	openaiResp := OpenAIResponse{
		ID:      anthropicResp.ID,
		Object:  "chat.completion",
		Model:   anthropicResp.Model,
		Choices: []OpenAIChoice{
			{
				Index: 0,
				Message: OpenAIMessage{
					Role:       "assistant",
					Content:    textContent.String(),
					ToolCalls:  toolCalls,
				},
				FinishReason: "tool_calls",
			},
		},
	}

	return json.Marshal(openaiResp)
}

// convertToOpenClaw 转换为 OpenClaw 格式
func (c *ToolCallConverter) convertToOpenClaw(resp *OpenAIResponse) ([]byte, error) {
	message := resp.Choices[0].Message

	var result strings.Builder

	// 添加文本内容
	if message.Content != "" {
		result.WriteString(message.Content)
		result.WriteString("\n\n")
	}

	// 添加工具调用
	for i, tc := range message.ToolCalls {
		result.WriteString(fmt.Sprintf("[工具调用 %d]\n", i+1))
		result.WriteString(fmt.Sprintf("ID: %s\n", tc.ID))
		result.WriteString(fmt.Sprintf("函数: %s\n", tc.Function.Name))

		// 格式化参数
		var args map[string]interface{}
		if err := json.Unmarshal(tc.Function.Arguments, &args); err == nil {
			argsJSON, _ := json.MarshalIndent(args, "  ", "  ")
			result.WriteString(fmt.Sprintf("参数: %s\n", string(argsJSON)))
		}
		result.WriteString("\n")
	}

	return []byte(result.String()), nil
}

// convertToAnthropic 转换为 Anthropic 格式
func (c *ToolCallConverter) convertToAnthropic(resp *OpenAIResponse) ([]byte, error) {
	message := resp.Choices[0].Message

	var content []AnthropicContentBlock

	// 添加文本内容
	if message.Content != "" {
		content = append(content, AnthropicContentBlock{
			Type: "text",
			Text: message.Content,
		})
	}

	// 添加工具调用
	for _, tc := range message.ToolCalls {
		var args map[string]interface{}
		json.Unmarshal(tc.Function.Arguments, &args)

		content = append(content, AnthropicContentBlock{
			Type:  "tool_use",
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: args,
		})
	}

	anthropicResp := map[string]interface{}{
		"id":           resp.ID,
		"type":         "message",
		"role":         "assistant",
		"content":      content,
		"model":        resp.Model,
		"stop_reason":  resp.Choices[0].FinishReason,
	}

	return json.Marshal(anthropicResp)
}

// ExtractToolCalls 从消息中提取工具调用
func ExtractToolCalls(message map[string]interface{}) ([]ToolCall, error) {
	toolCallsInterface, ok := message["tool_calls"]
	if !ok {
		return nil, nil // 没有工具调用
	}

	toolCallsArray, ok := toolCallsInterface.([]interface{})
	if !ok {
		return nil, fmt.Errorf("tool_calls is not an array")
	}

	var toolCalls []ToolCall
	for _, tcInterface := range toolCallsArray {
		tcMap, ok := tcInterface.(map[string]interface{})
		if !ok {
			continue
		}

		id, _ := tcMap["id"].(string)
		tcType, _ := tcMap["type"].(string)

		funcMap, ok := tcMap["function"].(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := funcMap["name"].(string)
		argsInterface, _ := funcMap["arguments"]

		arguments, _ := json.Marshal(argsInterface)

		toolCalls = append(toolCalls, ToolCall{
			ID:   id,
			Type: tcType,
			Function: FunctionCall{
				Name:      name,
				Arguments: json.RawMessage(arguments),
			},
		})
	}

	return toolCalls, nil
}

// FormatToolCallResult 格式化工具调用结果
func FormatToolCallResult(result *ToolCallResult, format string) ([]byte, error) {
	switch format {
	case "openai":
		return json.Marshal(map[string]interface{}{
			"tool_call_id": result.ToolCallID,
			"role":         "tool",
			"content":      result.Content,
		})
	case "openclaw":
		return []byte(fmt.Sprintf("[工具结果]\nID: %s\n内容: %s\n",
			result.ToolCallID, result.Content)), nil
	default:
		return json.Marshal(result)
	}
}
