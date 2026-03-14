package converter

import (
	"encoding/json"
	"testing"
)

// DeepSeek 格式解析测试
func TestDeepSeekParser_Parse(t *testing.T) {
	config := &ConverterConfig{
		Templates: TemplatesConfig{
			Message:     "%s",
			StreamChunk: "%s",
		},
		IncludeUsage: true,
	}

	parser := NewDeepSeekParser(config)

	// DeepSeek 响应示例
	input := `{
		"choices": [{
			"message": {
				"role": "assistant",
				"content": [
					{"type": "text", "text": "Hello, "},
					{"type": "text", "text": "world!"}
				]
			},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 20,
			"total_tokens": 30
		}
	}`

	// 解析
	resp, err := parser.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// 验证
	if len(resp.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(resp.Choices))
	}

	if resp.Usage.PromptTokens != 10 {
		t.Errorf("Expected 10 prompt tokens, got %d", resp.Usage.PromptTokens)
	}

	// 提取文本
	text, err := parser.ExtractText(resp)
	if err != nil {
		t.Fatalf("ExtractText failed: %v", err)
	}

	expectedText := "Hello, world!"
	if text != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, text)
	}
}

// 格式转换测试
func TestDeepSeekConverter_Convert(t *testing.T) {
	config := &ConverterConfig{
		Templates: TemplatesConfig{
			Message:     "%s",
			StreamChunk: "%s",
		},
		IncludeUsage: true,
	}

	converter := NewDeepSeekConverter(config)

	// DeepSeek 响应示例
	input := `{
		"choices": [{
			"message": {
				"role": "assistant",
				"content": [
					{"type": "text", "text": "AI response"}
				]
			},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 5,
			"completion_tokens": 10,
			"total_tokens": 15
		}
	}`

	// 转换
	output, err := converter.Convert([]byte(input))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	outputStr := string(output)

	// 验证内容包含文本和使用量
	if !contains(outputStr, "AI response") {
		t.Errorf("Output should contain 'AI response', got: %s", outputStr)
	}

	if !contains(outputStr, "usage:") {
		t.Errorf("Output should contain usage info, got: %s", outputStr)
	}

	if !contains(outputStr, "prompt=5") {
		t.Errorf("Output should contain prompt=5, got: %s", outputStr)
	}
}

// 空内容测试
func TestDeepSeekConverter_EmptyContent(t *testing.T) {
	config := &ConverterConfig{
		Templates: TemplatesConfig{
			Message:     "%s",
			StreamChunk: "%s",
		},
	}

	converter := NewDeepSeekConverter(config)

	// 空内容响应
	input := `{
		"choices": [{
			"message": {
				"role": "assistant",
				"content": []
			},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 0,
			"completion_tokens": 0,
			"total_tokens": 0
		}
	}`

	output, err := converter.Convert([]byte(input))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	outputStr := string(output)
	if outputStr != "" {
		t.Errorf("Expected empty output, got: %s", outputStr)
	}
}

// NewConverter 测试
func TestNewConverter(t *testing.T) {
	tests := []struct {
		name        string
		inputFormat string
		expectError bool
	}{
		{"DeepSeek format", "deepseek", false},
		{"OpenAI format", "openai-json", false},
		{"Unknown format defaults to DeepSeek", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ConverterConfig{
				InputFormat: tt.inputFormat,
				Templates: TemplatesConfig{
					Message: "%s",
				},
			}

			cvt, err := NewConverter(config)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if cvt == nil && !tt.expectError {
				t.Error("Expected converter but got nil")
			}
		})
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && (s[:len(substr)] == substr ||
			contains(s[1:], substr)))
}

// 测试 DeepSeek 格式的序列化/反序列化
func TestDeepSeekResponseSerialization(t *testing.T) {
	original := DeepSeekResponse{
		Choices: []DeepSeekChoice{
			{
				Message: DeepSeekMessage{
					Role: "assistant",
					Content: []DeepSeekContentItem{
						{Type: "text", Text: "Test"},
					},
				},
				FinishReason: "stop",
			},
		},
		Usage: DeepSeekUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed DeepSeekResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(parsed.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(parsed.Choices))
	}

	if parsed.Choices[0].Message.Role != "assistant" {
		t.Errorf("Expected role 'assistant', got '%s'", parsed.Choices[0].Message.Role)
	}
}
