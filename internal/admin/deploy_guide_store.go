package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// DeployGuideStore 部署指南数据存储
type DeployGuideStore struct {
	db *sqlx.DB
}

// NewDeployGuideStore 创建部署指南存储
func NewDeployGuideStore(db *sqlx.DB) *DeployGuideStore {
	return &DeployGuideStore{db: db}
}

// DeployGuide 部署指南配置
type DeployGuide struct {
	ID               string      `json:"id" db:"id"`
	FrameworkID      string      `json:"framework_id" db:"framework_id"`
	ModelID          string      `json:"model_id" db:"model_id"`
	Name             string      `json:"name" db:"name"`
	Alias            string      `json:"alias" db:"alias"`
	InstallCmd       string      `json:"install_cmd" db:"install_cmd"`
	StartCmd         string      `json:"start_cmd" db:"start_cmd"`
	Params           JSONMap     `json:"params" db:"params"`
	Features         JSONMap     `json:"features" db:"features"`
	Requirements     JSONMap     `json:"requirements" db:"requirements"`
	APIPort          int         `json:"api_port" db:"api_port"`
	Tagline          string      `json:"tagline" db:"tagline"`
	Description      string      `json:"description" db:"description"`
	Badge            string      `json:"badge" db:"badge"`
	BadgeColor       string      `json:"badge_color" db:"badge_color"`
	AccentColor      string      `json:"accent_color" db:"accent_color"`
	Icon             string      `json:"icon" db:"icon"`
	ModelFamily      string      `json:"model_family" db:"model_family"`
	VRAMRequirement string      `json:"vram_requirement" db:"vram_requirement"`
	Precision        string      `json:"precision" db:"precision"`
	HFID             string      `json:"hf_id" db:"hf_id"`
	Steps            JSONMap     `json:"steps" db:"steps"`
	DisplayOrder     int         `json:"display_order" db:"display_order"`
	IsActive         bool        `json:"is_active" db:"is_active"`
	CreatedAt        time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at" db:"updated_at"`
}

// JSONMap JSON 字段类型
type JSONMap map[string]interface{}

// Scan 实现 sql.Scanner 接口
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}
	return json.Unmarshal(bytes, j)
}

// List 列出所有部署指南
func (s *DeployGuideStore) List(ctx context.Context, activeOnly bool) ([]DeployGuide, error) {
	query := `SELECT * FROM deploy_guides`
	if activeOnly {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY framework_id, display_order`

	var guides []DeployGuide
	err := s.db.SelectContext(ctx, &guides, query)
	if err != nil {
		return nil, fmt.Errorf("list deploy guides: %w", err)
	}

	return guides, nil
}

// GetByID 根据 ID 获取部署指南
func (s *DeployGuideStore) GetByID(ctx context.Context, id string) (*DeployGuide, error) {
	var guide DeployGuide
	err := s.db.GetContext(ctx, &guide, `SELECT * FROM deploy_guides WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("get deploy guide by id: %w", err)
	}
	return &guide, nil
}

// GetByFramework 获取指定框架的部署指南
func (s *DeployGuideStore) GetByFramework(ctx context.Context, frameworkID string) ([]DeployGuide, error) {
	var guides []DeployGuide
	err := s.db.SelectContext(ctx, &guides,
		`SELECT * FROM deploy_guides WHERE framework_id = $1 AND is_active = true ORDER BY display_order`,
		frameworkID)
	if err != nil {
		return nil, fmt.Errorf("get deploy guides by framework: %w", err)
	}
	return guides, nil
}

// GetFrameworks 获取所有框架列表（框架级别记录）
func (s *DeployGuideStore) GetFrameworks(ctx context.Context) ([]DeployGuide, error) {
	var guides []DeployGuide
	err := s.db.SelectContext(ctx, &guides,
		`SELECT * FROM deploy_guides WHERE model_id = '_framework' AND is_active = true ORDER BY framework_id`)
	if err != nil {
		return nil, fmt.Errorf("get frameworks: %w", err)
	}
	return guides, nil
}

// Create 创建部署指南
func (s *DeployGuideStore) Create(ctx context.Context, guide *DeployGuide) error {
	query := `
		INSERT INTO deploy_guides (
			framework_id, model_id, name, alias, install_cmd, start_cmd,
			params, features, requirements, api_port, tagline, description,
			badge, badge_color, accent_color, icon, model_family,
			vram_requirement, precision, hf_id, steps, display_order, is_active
		) VALUES (
			:framework_id, :model_id, :name, :alias, :install_cmd, :start_cmd,
			:params, :features, :requirements, :api_port, :tagline, :description,
			:badge, :badge_color, :accent_color, :icon, :model_family,
			:vram_requirement, :precision, :hf_id, :steps, :display_order, :is_active
		) RETURNING id, created_at, updated_at
	`

	rows, err := s.db.NamedQueryContext(ctx, query, guide)
	if err != nil {
		return fmt.Errorf("create deploy guide: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&guide.ID, &guide.CreatedAt, &guide.UpdatedAt)
	}

	return fmt.Errorf("failed to create deploy guide: no rows returned")
}

// Update 更新部署指南
func (s *DeployGuideStore) Update(ctx context.Context, guide *DeployGuide) error {
	query := `
		UPDATE deploy_guides SET
			name = :name,
			alias = :alias,
			install_cmd = :install_cmd,
			start_cmd = :start_cmd,
			params = :params,
			features = :features,
			requirements = :requirements,
			api_port = :api_port,
			tagline = :tagline,
			description = :description,
			badge = :badge,
			badge_color = :badge_color,
			accent_color = :accent_color,
			icon = :icon,
			model_family = :model_family,
			vram_requirement = :vram_requirement,
			precision = :precision,
			hf_id = :hf_id,
			steps = :steps,
			display_order = :display_order,
			is_active = :is_active,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = :id
		RETURNING updated_at
	`

	rows, err := s.db.NamedQueryContext(ctx, query, guide)
	if err != nil {
		return fmt.Errorf("update deploy guide: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&guide.UpdatedAt)
	}

	return fmt.Errorf("failed to update deploy guide: no rows returned")
}

// Delete 删除部署指南
func (s *DeployGuideStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM deploy_guides WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete deploy guide: %w", err)
	}
	return nil
}
