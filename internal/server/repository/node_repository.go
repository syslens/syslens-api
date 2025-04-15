package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/syslens/syslens-api/internal/common/utils"
	"github.com/syslens/syslens-api/internal/server/storage"
)

// NodeStatus 定义节点状态类型
type NodeStatus string

const (
	NodeStatusPending  NodeStatus = "pending"
	NodeStatusActive   NodeStatus = "active"
	NodeStatusInactive NodeStatus = "inactive"
)

// NodeType 定义节点类型
type NodeType string

const (
	NodeTypeAgent        NodeType = "agent"
	NodeTypeFixedService NodeType = "fixed-service"
)

// Node 表示节点实体
type Node struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	AuthTokenHash      string         `json:"-"` // 不在JSON中暴露
	EncryptedAuthToken string         `json:"-"` // 加密存储的原始令牌，不在JSON中暴露
	Labels             map[string]any `json:"labels"`
	Configuration      map[string]any `json:"configuration,omitempty"`
	Type               NodeType       `json:"type"`
	Status             NodeStatus     `json:"status"`
	GroupID            sql.NullString `json:"group_id,omitempty"`
	ServiceID          sql.NullString `json:"service_id,omitempty"`
	Description        sql.NullString `json:"description,omitempty"`
	RegisteredAt       sql.NullTime   `json:"registered_at,omitempty"`
	LastActiveAt       sql.NullTime   `json:"last_active_at,omitempty"`
	CreatedAt          time.Time      `json:"created_time"`
	UpdatedAt          time.Time      `json:"updated_time"`
}

// NodeRepository 定义节点仓库接口
type NodeRepository interface {
	// Create 创建新节点
	Create(ctx context.Context, node *Node) error

	// GetByID 根据ID获取节点
	GetByID(ctx context.Context, id string) (*Node, error)

	// GetAll 获取所有节点
	GetAll(ctx context.Context) ([]*Node, error)

	// GetByStatus 根据状态获取节点
	GetByStatus(ctx context.Context, status NodeStatus) ([]*Node, error)

	// GetByGroupID 根据分组ID获取节点
	GetByGroupID(ctx context.Context, groupID string) ([]*Node, error)

	// GetByServiceID 根据服务ID获取节点
	GetByServiceID(ctx context.Context, serviceID string) ([]*Node, error)

	// Update 更新节点
	Update(ctx context.Context, node *Node) error

	// UpdateStatus 更新节点状态
	UpdateStatus(ctx context.Context, id string, status NodeStatus) error

	// UpdateLastActiveAt 更新节点最后活跃时间
	UpdateLastActiveAt(ctx context.Context, id string, lastActiveAt time.Time) error

	// Delete 删除节点
	Delete(ctx context.Context, id string) error

	// ValidateNodeToken 验证节点令牌
	ValidateNodeToken(ctx context.Context, id string, token string) (bool, error)

	// UpdateConfiguration 更新节点配置
	UpdateConfiguration(ctx context.Context, id string, configuration map[string]any) error

	// FindByToken 通过令牌查找节点
	FindByToken(ctx context.Context, token string) (*Node, error)
}

// PostgresNodeRepository 实现基于PostgreSQL的节点仓库
type PostgresNodeRepository struct {
	db *storage.PostgresDB
}

// NewPostgresNodeRepository 创建新的PostgreSQL节点仓库
func NewPostgresNodeRepository(db *storage.PostgresDB) *PostgresNodeRepository {
	return &PostgresNodeRepository{
		db: db,
	}
}

// Create 创建新节点
func (r *PostgresNodeRepository) Create(ctx context.Context, node *Node) error {
	// 将labels转换为JSONB
	labelsJSON, err := json.Marshal(node.Labels)
	if err != nil {
		return fmt.Errorf("序列化节点标签失败: %w", err)
	}

	// 将configuration转换为JSONB
	var configJSON []byte
	if node.Configuration != nil {
		configJSON, err = json.Marshal(node.Configuration)
		if err != nil {
			return fmt.Errorf("序列化节点配置失败: %w", err)
		}
	} else {
		// 确保配置为空时使用有效的空JSON对象，而不是null
		configJSON = []byte("{}")
	}

	query := `
		INSERT INTO nodes (
			id, name, auth_token_hash, encrypted_auth_token, labels, configuration, type, status, 
			group_id, service_id, description, registered_at, last_active_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		) RETURNING created_time, updated_time
	`

	// 执行插入
	err = r.db.QueryRowContext(
		ctx,
		query,
		node.ID,
		node.Name,
		node.AuthTokenHash,
		node.EncryptedAuthToken,
		labelsJSON,
		configJSON,
		node.Type,
		node.Status,
		node.GroupID,
		node.ServiceID,
		node.Description,
		node.RegisteredAt,
		node.LastActiveAt,
	).Scan(&node.CreatedAt, &node.UpdatedAt)

	if err != nil {
		return fmt.Errorf("创建节点失败: %w", err)
	}

	return nil
}

// GetByID 根据ID获取节点
func (r *PostgresNodeRepository) GetByID(ctx context.Context, id string) (*Node, error) {
	query := `
		SELECT 
			id, name, auth_token_hash, encrypted_auth_token, labels, configuration, type, status, 
			group_id, service_id, description, registered_at, last_active_at,
			created_time, updated_time
		FROM nodes
		WHERE id = $1
	`

	var node Node
	var labelsJSON []byte
	var configJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&node.ID,
		&node.Name,
		&node.AuthTokenHash,
		&node.EncryptedAuthToken,
		&labelsJSON,
		&configJSON,
		&node.Type,
		&node.Status,
		&node.GroupID,
		&node.ServiceID,
		&node.Description,
		&node.RegisteredAt,
		&node.LastActiveAt,
		&node.CreatedAt,
		&node.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 未找到节点
		}
		return nil, fmt.Errorf("获取节点失败: %w", err)
	}

	// 解析标签JSON
	if err := json.Unmarshal(labelsJSON, &node.Labels); err != nil {
		return nil, fmt.Errorf("解析节点标签失败: %w", err)
	}

	// 解析配置JSON
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &node.Configuration); err != nil {
			return nil, fmt.Errorf("解析节点配置失败: %w", err)
		}
	}

	return &node, nil
}

// GetAll 获取所有节点
func (r *PostgresNodeRepository) GetAll(ctx context.Context) ([]*Node, error) {
	query := `
		SELECT 
			id, name, auth_token_hash, encrypted_auth_token, labels, configuration, type, status, 
			group_id, service_id, description, registered_at, last_active_at,
			created_time, updated_time
		FROM nodes
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询所有节点失败: %w", err)
	}
	defer rows.Close()

	return r.scanNodes(rows)
}

// GetByStatus 根据状态获取节点
func (r *PostgresNodeRepository) GetByStatus(ctx context.Context, status NodeStatus) ([]*Node, error) {
	query := `
		SELECT 
			id, name, auth_token_hash, encrypted_auth_token, labels, configuration, type, status, 
			group_id, service_id, description, registered_at, last_active_at,
			created_time, updated_time
		FROM nodes
		WHERE status = $1
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("查询状态为 %s 的节点失败: %w", status, err)
	}
	defer rows.Close()

	return r.scanNodes(rows)
}

// GetByGroupID 根据分组ID获取节点
func (r *PostgresNodeRepository) GetByGroupID(ctx context.Context, groupID string) ([]*Node, error) {
	query := `
		SELECT 
			id, name, auth_token_hash, encrypted_auth_token, labels, configuration, type, status, 
			group_id, service_id, description, registered_at, last_active_at,
			created_time, updated_time
		FROM nodes
		WHERE group_id = $1
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("查询分组 %s 的节点失败: %w", groupID, err)
	}
	defer rows.Close()

	return r.scanNodes(rows)
}

// GetByServiceID 根据服务ID获取节点
func (r *PostgresNodeRepository) GetByServiceID(ctx context.Context, serviceID string) ([]*Node, error) {
	query := `
		SELECT 
			id, name, auth_token_hash, encrypted_auth_token, labels, configuration, type, status, 
			group_id, service_id, description, registered_at, last_active_at,
			created_time, updated_time
		FROM nodes
		WHERE service_id = $1
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, serviceID)
	if err != nil {
		return nil, fmt.Errorf("查询服务 %s 的节点失败: %w", serviceID, err)
	}
	defer rows.Close()

	return r.scanNodes(rows)
}

// Update 更新节点
func (r *PostgresNodeRepository) Update(ctx context.Context, node *Node) error {
	// 将labels转换为JSONB
	labelsJSON, err := json.Marshal(node.Labels)
	if err != nil {
		return fmt.Errorf("序列化节点标签失败: %w", err)
	}

	// 将configuration转换为JSONB
	var configJSON []byte
	if node.Configuration != nil {
		configJSON, err = json.Marshal(node.Configuration)
		if err != nil {
			return fmt.Errorf("序列化节点配置失败: %w", err)
		}
	} else {
		// 确保配置为空时使用有效的空JSON对象，而不是null
		configJSON = []byte("{}")
	}

	query := `
		UPDATE nodes
		SET 
			name = $2,
			auth_token_hash = $3,
			encrypted_auth_token = $4,
			labels = $5,
			configuration = $6,
			type = $7,
			status = $8,
			group_id = $9,
			service_id = $10,
			description = $11,
			registered_at = $12,
			last_active_at = $13
		WHERE id = $1
		RETURNING updated_time
	`

	err = r.db.QueryRowContext(
		ctx,
		query,
		node.ID,
		node.Name,
		node.AuthTokenHash,
		node.EncryptedAuthToken,
		labelsJSON,
		configJSON,
		node.Type,
		node.Status,
		node.GroupID,
		node.ServiceID,
		node.Description,
		node.RegisteredAt,
		node.LastActiveAt,
	).Scan(&node.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("节点 %s 不存在", node.ID)
		}
		return fmt.Errorf("更新节点失败: %w", err)
	}

	return nil
}

// UpdateStatus 更新节点状态
func (r *PostgresNodeRepository) UpdateStatus(ctx context.Context, id string, status NodeStatus) error {
	query := `
		UPDATE nodes
		SET status = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("更新节点状态失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("节点 %s 不存在", id)
	}

	return nil
}

// UpdateLastActiveAt 更新节点最后活跃时间
func (r *PostgresNodeRepository) UpdateLastActiveAt(ctx context.Context, id string, lastActiveAt time.Time) error {
	query := `
		UPDATE nodes
		SET last_active_at = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, lastActiveAt)
	if err != nil {
		return fmt.Errorf("更新节点最后活跃时间失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("节点 %s 不存在", id)
	}

	return nil
}

// Delete 删除节点
func (r *PostgresNodeRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM nodes WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("删除节点失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("节点 %s 不存在", id)
	}

	return nil
}

// ValidateNodeToken 验证节点令牌
func (r *PostgresNodeRepository) ValidateNodeToken(ctx context.Context, id string, token string) (bool, error) {
	query := `
		SELECT auth_token_hash
		FROM nodes
		WHERE id = $1
	`

	var storedTokenHash string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&storedTokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // 节点不存在
		}
		return false, fmt.Errorf("获取节点令牌失败: %w", err)
	}

	// 使用ComparePasswordAndHash函数验证token是否匹配
	isValid := utils.ComparePasswordAndHash(token, storedTokenHash)
	return isValid, nil
}

// scanNodes 扫描查询结果并返回节点列表
func (r *PostgresNodeRepository) scanNodes(rows *sql.Rows) ([]*Node, error) {
	var nodes []*Node

	for rows.Next() {
		var node Node
		var labelsJSON []byte
		var configJSON []byte

		err := rows.Scan(
			&node.ID,
			&node.Name,
			&node.AuthTokenHash,
			&node.EncryptedAuthToken,
			&labelsJSON,
			&configJSON,
			&node.Type,
			&node.Status,
			&node.GroupID,
			&node.ServiceID,
			&node.Description,
			&node.RegisteredAt,
			&node.LastActiveAt,
			&node.CreatedAt,
			&node.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("扫描节点行失败: %w", err)
		}

		// 解析标签JSON
		if err := json.Unmarshal(labelsJSON, &node.Labels); err != nil {
			return nil, fmt.Errorf("解析节点标签失败: %w", err)
		}

		// 解析配置JSON
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &node.Configuration); err != nil {
				return nil, fmt.Errorf("解析节点配置失败: %w", err)
			}
		}

		nodes = append(nodes, &node)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("迭代节点行失败: %w", err)
	}

	return nodes, nil
}

// UpdateConfiguration 更新节点配置
func (r *PostgresNodeRepository) UpdateConfiguration(ctx context.Context, id string, configuration map[string]any) error {
	// 将configuration转换为JSONB
	var configJSON []byte
	var err error

	if configuration != nil {
		configJSON, err = json.Marshal(configuration)
		if err != nil {
			return fmt.Errorf("序列化节点配置失败: %w", err)
		}
	} else {
		// 确保配置为空时使用有效的空JSON对象，而不是null
		configJSON = []byte("{}")
	}

	query := `
		UPDATE nodes
		SET configuration = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, configJSON)
	if err != nil {
		return fmt.Errorf("更新节点配置失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("节点 %s 不存在", id)
	}

	return nil
}

// FindByToken 通过令牌查找节点
func (r *PostgresNodeRepository) FindByToken(ctx context.Context, token string) (*Node, error) {
	// 由于token是哈希存储的，无法直接通过token查询
	// 需要获取所有节点，然后逐个验证token
	nodes, err := r.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取所有节点失败: %w", err)
	}

	for _, node := range nodes {
		// 验证token
		// 使用ComparePasswordAndHash函数验证token是否匹配
		isValid := utils.ComparePasswordAndHash(token, node.AuthTokenHash)
		if isValid {
			return node, nil // 找到匹配的节点
		}
	}

	return nil, nil // 未找到匹配的节点
}
