package converter

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RoleMapper 角色映射器
type RoleMapper struct {
	mappings        map[string]string
	openclawFormat  bool
	customMappings  map[string]string
}

// NewRoleMapper 创建角色映射器
func NewRoleMapper(openclawFormat bool) *RoleMapper {
	mapper := &RoleMapper{
		openclawFormat: openclawFormat,
		mappings:       make(map[string]string),
		customMappings: make(map[string]string),
	}

	// 默认角色映射
	mapper.setDefaultMappings()

	return mapper
}

// setDefaultMappings 设置默认角色映射
func (m *RoleMapper) setDefaultMappings() {
	// OpenAI 标准角色
	m.mappings["system"] = "system"
	m.mappings["user"] = "user"
	m.mappings["assistant"] = "assistant"
	m.mappings["tool"] = "tool"
	m.mappings["function"] = "tool"

	// 其他常见角色
	m.mappings["developer"] = "system"
	m.mappings["code"] = "assistant"
}

// AddCustomMapping 添加自定义角色映射
func (m *RoleMapper) AddCustomMapping(from, to string) {
	m.customMappings[from] = to
}

// MapRole 映射角色名称
func (m *RoleMapper) MapRole(role string) string {
	// 先检查自定义映射
	if to, ok := m.customMappings[role]; ok {
		return m.formatRole(to)
	}

	// 使用默认映射
	if to, ok := m.mappings[role]; ok {
		return m.formatRole(to)
	}

	// 未知角色，保持原样
	return m.formatRole(role)
}

// formatRole 格式化角色输出
func (m *RoleMapper) formatRole(role string) string {
	if !m.openclawFormat {
		return role
	}

	// OpenClaw 角色格式
	switch role {
	case "system":
		return "<role>system</role>"
	case "user":
		return "<role>user</role>"
	case "assistant":
		return "<role>assistant</role>"
	case "tool":
		return "<role>tool</role>"
	default:
		return fmt.Sprintf("<role>%s</role>", role)
	}
}

// ParseRole 从 OpenClaw 格式解析角色
func (m *RoleMapper) ParseRole(content string) string {
	if !m.openclawFormat {
		return content
	}

	// 解析 <role>xxx</role> 格式
	if strings.HasPrefix(content, "<role>") && strings.HasSuffix(content, "</role>") {
		role := strings.TrimPrefix(content, "<role>")
		role = strings.TrimSuffix(role, "</role>")
		return role
	}

	return content
}

// Message 消息结构
type Message struct {
	Role         string `json:"role"`
	Content      string `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID   string `json:"tool_call_id,omitempty"`
}

// MessageConverter 消息转换器
type MessageConverter struct {
	roleMapper *RoleMapper
}

// NewMessageConverter 创建消息转换器
func NewMessageConverter(openclawFormat bool) *MessageConverter {
	return &MessageConverter{
		roleMapper: NewRoleMapper(openclawFormat),
	}
}

// ConvertMessage 转换单条消息
func (c *MessageConverter) ConvertMessage(msg *Message) string {
	mappedRole := c.roleMapper.MapRole(msg.Role)

	var result strings.Builder

	// 添加角色标记
	if c.roleMapper.openclawFormat {
		result.WriteString(mappedRole)
		result.WriteString("\n")
	}

	// 添加内容
	if msg.Content != "" {
		result.WriteString(msg.Content)
	}

	// 添加工具调用
	if len(msg.ToolCalls) > 0 {
		for _, tc := range msg.ToolCalls {
			result.WriteString(fmt.Sprintf("\n<tool_call name=\"%s\">%s\n",
				tc.Function.Name, tc.Function.Arguments))
		}
	}

	return result.String()
}

// ConvertMessages 转换消息列表
func (c *MessageConverter) ConvertMessages(messages []Message) string {
	var result strings.Builder

	for i, msg := range messages {
		converted := c.ConvertMessage(&msg)
		result.WriteString(converted)

		// 消息之间添加分隔符
		if i < len(messages)-1 {
			result.WriteString("\n\n---\n\n")
		}
	}

	return result.String()
}

// ParseOpenAIMessage 解析 OpenAI 消息格式
func ParseOpenAIMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ParseOpenAIMessages 解析 OpenAI 消息列表
func ParseOpenAIMessages(data []byte) ([]Message, error) {
	var messages []Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// FormatMessagesForLLM 格式化消息供 LLM 使用
func FormatMessagesForLLM(messages []Message, format string) (string, error) {
	converter := NewMessageConverter(format == "openclaw")

	var result strings.Builder

	for _, msg := range messages {
		role := converter.roleMapper.MapRole(msg.Role)

		if format == "openclaw" {
			result.WriteString(fmt.Sprintf("%s\n%s\n\n", role, msg.Content))
		} else {
			// JSON 格式
			result.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content))
		}
	}

	return result.String(), nil
}

// ValidateRole 验证角色是否有效
func ValidateRole(role string) bool {
	validRoles := map[string]bool{
		"system":      true,
		"user":        true,
		"assistant":   true,
		"tool":        true,
		"function":    true,
		"developer":   true,
	}

	return validRoles[role]
}

// NormalizeRole 标准化角色名称
func NormalizeRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))

	// 处理变体
	roleMap := map[string]string{
		"function": "tool",
		"bot":      "assistant",
		"ai":       "assistant",
		"human":    "user",
	}

	if normalized, ok := roleMap[role]; ok {
		return normalized
	}

	return role
}
