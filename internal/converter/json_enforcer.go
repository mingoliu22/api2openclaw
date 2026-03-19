package converter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// JSONEnforceStrategy JSON 强制输出策略
type JSONEnforceStrategy string

const (
	// JSONEnforceReturnError 返回错误（200 + error_body）
	JSONEnforceReturnError JSONEnforceStrategy = "return_error"
	// JSONEnforceReturn500 返回 HTTP 500 错误
	JSONEnforceReturn500 JSONEnforceStrategy = "return_500"
)

// JSONEnforcerConfig JSON 强制输出配置
type JSONEnforcerConfig struct {
	// Strategy 兜底失败时的响应策略
	Strategy JSONEnforceStrategy `yaml:"json_enforce_strategy"`

	// StrictMode 严格模式：只有完全符合 JSON 规范才通过
	StrictMode bool `yaml:"json_strict_mode"`

	// AllowComments 是否允许 JSON 注释（非标准）
	AllowComments bool `yaml:"json_allow_comments"`

	// AllowTrailingComma 是否允许尾部逗号（非标准）
	AllowTrailingComma bool `yaml:"json_allow_trailing_comma"`
}

// DefaultJSONEnforcerConfig 默认配置
var DefaultJSONEnforcerConfig = &JSONEnforcerConfig{
	Strategy:           JSONEnforceReturnError,
	StrictMode:          true,
	AllowComments:       false,
	AllowTrailingComma:  false,
}

// JSONEnforcer JSON 强制输出器
type JSONEnforcer struct {
	config *JSONEnforcerConfig
}

// NewJSONEnforcer 创建 JSON 强制输出器
func NewJSONEnforcer(config *JSONEnforcerConfig) *JSONEnforcer {
	if config == nil {
		config = DefaultJSONEnforcerConfig
	}
	return &JSONEnforcer{config: config}
}

// EnforceJSONFormat 强制 JSON 输出（三级兜底）
// 返回：JSON 字符串，是否触发兜底，错误
func (e *JSONEnforcer) EnforceJSONFormat(response string) (string, bool, error) {
	// 一级兜底：直接解析为 JSON
	if cleaned, err := e.tryParseJSON(response); err == nil {
		return cleaned, false, nil
	}

	// 二级兜底：提取 ```json 代码块
	if extracted, err := e.extractCodeBlock(response); err == nil {
		if cleaned, err := e.tryParseJSON(extracted); err == nil {
			return cleaned, true, nil
		}
	}

	// 三级兜底：策略处理
	return "", true, &JSONEnforceError{
		Strategy: e.config.Strategy,
		RawContent: response,
		Message: "Unable to enforce JSON output after all fallback attempts",
	}
}

// EnforceWithSchema 使用 schema 验证 JSON 输出
func (e *JSONEnforcer) EnforceWithSchema(response string, schema *JSONSchema) (string, bool, error) {
	jsonStr, _, err := e.EnforceJSONFormat(response)
	if err != nil {
		return "", true, err
	}

	// 验证 schema
	if schema != nil {
		if err := schema.Validate(jsonStr); err != nil {
			return "", true, fmt.Errorf("schema validation failed: %w", err)
		}
	}

	return jsonStr, true, nil
}

// tryParseJSON 尝试解析 JSON
func (e *JSONEnforcer) tryParseJSON(s string) (string, error) {
	s = strings.TrimSpace(s)

	// 空字符串
	if s == "" {
		return "", fmt.Errorf("empty string")
	}

	// 预处理
	if e.config.AllowComments {
		s = removeJSONComments(s)
	}
	if e.config.AllowTrailingComma {
		s = removeTrailingCommas(s)
	}

	// 尝试解析
	var result interface{}
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// 重新序列化以规范化格式
	cleaned, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("re-marshal failed: %w", err)
	}

	return string(cleaned), nil
}

// extractCodeBlock 从响应中提取 ```json 代码块
func (e *JSONEnforcer) extractCodeBlock(response string) (string, error) {
	// 正则匹配 ```json ... ``` 或 ``` ... ```
	pattern := regexp.MustCompile("```(?:json)?\n?([\s\S]*?)```")
	matches := pattern.FindStringSubmatch(response)

	if len(matches) < 2 {
		return "", fmt.Errorf("no JSON code block found")
	}

	return strings.TrimSpace(matches[1]), nil
}

// JSONEnforceError JSON 强制输出错误
type JSONEnforceError struct {
	Strategy   JSONEnforceStrategy
	RawContent string
	Message    string
}

func (e *JSONEnforceError) Error() string {
	return e.Message
}

// ToHTTPStatus 转换为 HTTP 状态码
func (e *JSONEnforceError) ToHTTPStatus() int {
	switch e.Strategy {
	case JSONEnforceReturn500:
		return 500
	default:
		return 200
	}
}

// ToErrorResponse 转换为错误响应体
func (e *JSONEnforceError) ToErrorResponse() map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{
			"code":    "json_format_failed",
			"message": e.Message,
			"raw_preview": safePreview(e.RawContent, 200),
		},
	}
}

// safePreview 安全截取预览
func safePreview(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// === JSON Schema 支持 ===

// JSONSchema JSON Schema 定义
type JSONSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]*JSONSchema `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
	Items      *JSONSchema            `json:"items,omitempty"`
}

// Validate 验证 JSON 是否符合 schema
func (s *JSONSchema) Validate(jsonStr string) error {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return err
	}

	return s.validateValue(data)
}

func (s *JSONSchema) validateValue(value interface{}) error {
	switch s.Type {
	case "object":
		return s.validateObject(value)
	case "array":
		return s.validateArray(value)
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "number":
		switch value.(type) {
		case float64, float32, int, int64, int32:
		default:
			return fmt.Errorf("expected number, got %T", value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "null":
		if value != nil {
			return fmt.Errorf("expected null, got %v", value)
		}
	}
	return nil
}

func (s *JSONSchema) validateObject(value interface{}) error {
	obj, ok := value.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected object, got %T", value)
	}

	// 检查必填字段
	for _, req := range s.Required {
		if _, exists := obj[req]; !exists {
			return fmt.Errorf("missing required field: %s", req)
		}
	}

	// 验证每个属性
	for key, prop := range s.Properties {
		if val, exists := obj[key]; exists {
			if err := prop.validateValue(val); err != nil {
				return fmt.Errorf("field %s: %w", key, err)
			}
		}
	}

	return nil
}

func (s *JSONSchema) validateArray(value interface{}) error {
	arr, ok := value.([]interface{})
	if !ok {
		return fmt.Errorf("expected array, got %T", value)
	}

	if s.Items != nil {
		for i, item := range arr {
			if err := s.Items.validateValue(item); err != nil {
				return fmt.Errorf("array[%d]: %w", i, err)
			}
		}
	}

	return nil
}

// === 辅助函数 ===

// removeJSONComments 移除 JSON 注释
func removeJSONComments(s string) string {
	// 移除单行注释 //
	re := regexp.MustCompile(`//.*`)
	s = re.ReplaceAllString(s, "")

	// 移除多行注释 /* */
	re = regexp.MustCompile(`/\*[\s\S]*?\*/`)
	s = re.ReplaceAllString(s, "")

	return s
}

// removeTrailingCommas 移除尾部逗号
func removeTrailingCommas(s string) string {
	// 移除对象/数组中的尾部逗号：}, ], }, ]
	re := regexp.MustCompile(`,(\s*[}\]])`)
	return re.ReplaceAllString(s, "$1")
}

// IsValidJSON 快速检查字符串是否为有效 JSON
func IsValidJSON(s string) bool {
	var v interface{}
	return json.Unmarshal([]byte(s), &v) == nil
}
