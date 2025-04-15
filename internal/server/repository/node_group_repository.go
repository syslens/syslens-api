package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/syslens/syslens-api/internal/server/storage"
)

// NodeGroup 表示节点分组实体
type NodeGroup struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        sql.NullString `json:"type,omitempty"`
	Description sql.NullString `json:"description,omitempty"`
	CreatedAt   time.Time      `json:"created_time"`
	UpdatedAt   time.Time      `json:"updated_time"`
}

// NodeGroupRepository 定义节点分组仓库接口
type NodeGroupRepository interface {
	// Create 创建新分组
	Create(ctx context.Context, group *NodeGroup) error

	// GetByID 根据ID获取分组
	GetByID(ctx context.Context, id string) (*NodeGroup, error)

	// GetByName 根据名称获取分组
	GetByName(ctx context.Context, name string) (*NodeGroup, error)

	// GetAll 获取所有分组
	GetAll(ctx context.Context) ([]*NodeGroup, error)

	// GetByType 根据类型获取分组
	GetByType(ctx context.Context, groupType string) ([]*NodeGroup, error)

	// Update 更新分组
	Update(ctx context.Context, group *NodeGroup) error

	// Delete 删除分组
	Delete(ctx context.Context, id string) error
}

// PostgresNodeGroupRepository 实现基于PostgreSQL的节点分组仓库
type PostgresNodeGroupRepository struct {
	db *storage.PostgresDB
}

// NewPostgresNodeGroupRepository 创建新的PostgreSQL节点分组仓库
func NewPostgresNodeGroupRepository(db *storage.PostgresDB) *PostgresNodeGroupRepository {
	return &PostgresNodeGroupRepository{
		db: db,
	}
}

// Create 创建新分组
func (r *PostgresNodeGroupRepository) Create(ctx context.Context, group *NodeGroup) error {
	query := `
		INSERT INTO node_groups (
			id, name, type, description
		) VALUES (
			$1, $2, $3, $4
		) RETURNING created_time, updated_time
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		group.ID,
		group.Name,
		group.Type,
		group.Description,
	).Scan(&group.CreatedAt, &group.UpdatedAt)

	if err != nil {
		return fmt.Errorf("创建节点分组失败: %w", err)
	}

	return nil
}

// GetByID 根据ID获取分组
func (r *PostgresNodeGroupRepository) GetByID(ctx context.Context, id string) (*NodeGroup, error) {
	query := `
		SELECT 
			id, name, type, description, created_time, updated_time
		FROM node_groups
		WHERE id = $1
	`

	var group NodeGroup
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&group.ID,
		&group.Name,
		&group.Type,
		&group.Description,
		&group.CreatedAt,
		&group.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 未找到分组
		}
		return nil, fmt.Errorf("获取节点分组失败: %w", err)
	}

	return &group, nil
}

// GetByName 根据名称获取分组
func (r *PostgresNodeGroupRepository) GetByName(ctx context.Context, name string) (*NodeGroup, error) {
	query := `
		SELECT 
			id, name, type, description, created_time, updated_time
		FROM node_groups
		WHERE name = $1
	`

	var group NodeGroup
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&group.ID,
		&group.Name,
		&group.Type,
		&group.Description,
		&group.CreatedAt,
		&group.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 未找到分组
		}
		return nil, fmt.Errorf("获取节点分组失败: %w", err)
	}

	return &group, nil
}

// GetAll 获取所有分组
func (r *PostgresNodeGroupRepository) GetAll(ctx context.Context) ([]*NodeGroup, error) {
	query := `
		SELECT 
			id, name, type, description, created_time, updated_time
		FROM node_groups
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询所有节点分组失败: %w", err)
	}
	defer rows.Close()

	return r.scanGroups(rows)
}

// GetByType 根据类型获取分组
func (r *PostgresNodeGroupRepository) GetByType(ctx context.Context, groupType string) ([]*NodeGroup, error) {
	query := `
		SELECT 
			id, name, type, description, created_time, updated_time
		FROM node_groups
		WHERE type = $1
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, groupType)
	if err != nil {
		return nil, fmt.Errorf("查询类型为 %s 的节点分组失败: %w", groupType, err)
	}
	defer rows.Close()

	return r.scanGroups(rows)
}

// Update 更新分组
func (r *PostgresNodeGroupRepository) Update(ctx context.Context, group *NodeGroup) error {
	query := `
		UPDATE node_groups
		SET 
			name = $2,
			type = $3,
			description = $4
		WHERE id = $1
		RETURNING updated_time
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		group.ID,
		group.Name,
		group.Type,
		group.Description,
	).Scan(&group.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("节点分组 %s 不存在", group.ID)
		}
		return fmt.Errorf("更新节点分组失败: %w", err)
	}

	return nil
}

// Delete 删除分组
func (r *PostgresNodeGroupRepository) Delete(ctx context.Context, id string) error {
	// 开始事务
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 首先更新引用此分组的节点
	updateNodesQuery := `
		UPDATE nodes
		SET group_id = NULL
		WHERE group_id = $1
	`

	_, err = tx.ExecContext(ctx, updateNodesQuery, id)
	if err != nil {
		return fmt.Errorf("更新引用分组的节点失败: %w", err)
	}

	// 然后删除分组
	deleteQuery := `DELETE FROM node_groups WHERE id = $1`
	result, err := tx.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return fmt.Errorf("删除节点分组失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("节点分组 %s 不存在", id)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// scanGroups 扫描查询结果并返回分组列表
func (r *PostgresNodeGroupRepository) scanGroups(rows *sql.Rows) ([]*NodeGroup, error) {
	var groups []*NodeGroup

	for rows.Next() {
		var group NodeGroup
		err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.Type,
			&group.Description,
			&group.CreatedAt,
			&group.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("扫描节点分组行失败: %w", err)
		}

		groups = append(groups, &group)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("迭代节点分组行失败: %w", err)
	}

	return groups, nil
}
