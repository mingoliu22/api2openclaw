package models

import "time"

// ModelConfig 模型配置
type ModelConfig struct {
	ID              int        `json:"id" db:"id"`
	Name            string     `json:"name" db:"name"`
	BackendGroup    []string   `json:"backend_group" db:"backend_group"`
	RoutingStrategy string     `json:"routing_strategy" db:"routing_strategy"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`

	// v0.3.0: 模型能力字段
	SupportsStreaming  bool   `json:"supports_streaming" db:"supports_streaming"`
	SupportsToolUse    bool   `json:"supports_tool_use" db:"supports_tool_use"`
	SupportsJSONMode   bool   `json:"supports_json_mode" db:"supports_json_mode"`
	ContextWindow      int    `json:"context_window" db:"context_window"`
	ModelFamily        string `json:"model_family" db:"model_family"`
}

// RoutingStrategy 路由策略类型
type RoutingStrategy string

const (
	RoutingStrategyDirect         RoutingStrategy = "direct"
	RoutingStrategyRoundRobin     RoutingStrategy = "round-robin"
	RoutingStrategyLeastConnections RoutingStrategy = "least-connections"
	RoutingStrategyRandom         RoutingStrategy = "random"
)
