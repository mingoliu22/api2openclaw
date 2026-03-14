package converter

import (
	"encoding/json"
	"fmt"
	"io"
)

// Converter 格式转换器接口
type Converter interface {
	// Convert 转换完整响应
	Convert(data []byte) ([]byte, error)

	// ConvertStream 转换流式响应
	ConvertStream(r io.Reader, w io.Writer) error
}

// ConverterConfig 转换器配置
type ConverterConfig struct {
	// InputFormat 输入格式: deepseek, openai-json
	InputFormat string `yaml:"input_format"`

	// OutputFormat 输出格式: openclaw, json
	OutputFormat string `yaml:"output_format"`

	// Templates 输出模板
	Templates TemplatesConfig `yaml:"templates"`

	// IncludeUsage 是否包含使用量信息
	IncludeUsage bool `yaml:"include_usage"`
}

// TemplatesConfig 模板配置
type TemplatesConfig struct {
	// Message 消息模板
	Message string `yaml:"message"`

	// StreamChunk 流式分块模板
	StreamChunk string `yaml:"stream_chunk"`
}

// NewConverter 创建转换器
func NewConverter(config *ConverterConfig) (Converter, error) {
	switch config.InputFormat {
	case "deepseek":
		return NewDeepSeekConverter(config), nil
	case "openai-json":
		return NewOpenAIConverter(config), nil
	default:
		return NewDeepSeekConverter(config), nil // 默认使用 DeepSeek
	}
}

// DeepSeekConverter DeepSeek 转换器
type DeepSeekConverter struct {
	parser *DeepSeekParser
	config *ConverterConfig
}

// NewDeepSeekConverter 创建 DeepSeek 转换器
func NewDeepSeekConverter(config *ConverterConfig) *DeepSeekConverter {
	return &DeepSeekConverter{
		parser: NewDeepSeekParser(config),
		config: config,
	}
}

// Convert 转换完整响应
func (c *DeepSeekConverter) Convert(data []byte) ([]byte, error) {
	return c.parser.ConvertToOpenClaw(data)
}

// ConvertStream 转换流式响应
func (c *DeepSeekConverter) ConvertStream(r io.Reader, w io.Writer) error {
	return c.parser.ConvertStream(r, w)
}

// OpenAIConverter OpenAI 格式转换器
type OpenAIConverter struct {
	config *ConverterConfig
}

// NewOpenAIConverter 创建 OpenAI 转换器
func NewOpenAIConverter(config *ConverterConfig) *OpenAIConverter {
	return &OpenAIConverter{config: config}
}

// Convert 转换完整响应
func (c *OpenAIConverter) Convert(data []byte) ([]byte, error) {
	// OpenAI 标准格式直接返回 content 字段
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return []byte{}, nil
	}

	content := resp.Choices[0].Message.Content
	template := c.config.Templates.Message
	output := fmt.Sprintf(template, content)

	return []byte(output), nil
}

// ConvertStream 转换流式响应
func (c *OpenAIConverter) ConvertStream(r io.Reader, w io.Writer) error {
	decoder := json.NewDecoder(r)

	for {
		var chunk map[string]interface{}
		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// OpenAI 流式格式: choices[0].delta.content
		if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if delta, ok := choice["delta"].(map[string]interface{}); ok {
					if content, ok := delta["content"].(string); ok {
						fmt.Fprintf(w, c.config.Templates.StreamChunk, content)
					}
				}
			}
		}
	}

	return nil
}
