package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/syslens/syslens-api/internal/server/storage"
)

// Service 表示服务实体
type Service struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     sql.NullString `json:"description,omitempty"`
	CriticalMetrics map[string]any `json:"critical_metrics,omitempty"`
	CreatedAt       time.Time      `json:"created_time"`
	UpdatedAt       time.Time      `json:"updated_time"`
}

// ServiceRepository 定义服务仓库接口
type ServiceRepository interface {
	// Create 创建新服务
	Create(ctx context.Context, service *Service) error

	// GetByID 根据ID获取服务
	GetByID(ctx context.Context, id string) (*Service, error)

	// GetByName 根据名称获取服务
	GetByName(ctx context.Context, name string) (*Service, error)

	// GetAll 获取所有服务
	GetAll(ctx context.Context) ([]*Service, error)

	// Update 更新服务
	Update(ctx context.Context, service *Service) error

	// Delete 删除服务
	Delete(ctx context.Context, id string) error
}

// PostgresServiceRepository 实现基于PostgreSQL的服务仓库
type PostgresServiceRepository struct {
	db *storage.PostgresDB
}

// NewPostgresServiceRepository 创建新的PostgreSQL服务仓库
func NewPostgresServiceRepository(db *storage.PostgresDB) *PostgresServiceRepository {
	return &PostgresServiceRepository{
		db: db,
	}
}

// Create 创建新服务
func (r *PostgresServiceRepository) Create(ctx context.Context, service *Service) error {
	// 将critical_metrics转换为JSONB
	criticalMetricsJSON, err := json.Marshal(service.CriticalMetrics)
	if err != nil {
		return fmt.Errorf("序列化服务关键指标失败: %w", err)
	}

	query := `
		INSERT INTO services (
			id, name, description, critical_metrics
		) VALUES (
			$1, $2, $3, $4
		) RETURNING created_time, updated_time
	`

	err = r.db.QueryRowContext(
		ctx,
		query,
		service.ID,
		service.Name,
		service.Description,
		criticalMetricsJSON,
	).Scan(&service.CreatedAt, &service.UpdatedAt)

	if err != nil {
		return fmt.Errorf("创建服务失败: %w", err)
	}

	return nil
}

// GetByID 根据ID获取服务
func (r *PostgresServiceRepository) GetByID(ctx context.Context, id string) (*Service, error) {
	query := `
		SELECT 
			id, name, description, critical_metrics, created_time, updated_time
		FROM services
		WHERE id = $1
	`

	var service Service
	var criticalMetricsJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&service.ID,
		&service.Name,
		&service.Description,
		&criticalMetricsJSON,
		&service.CreatedAt,
		&service.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 未找到服务
		}
		return nil, fmt.Errorf("获取服务失败: %w", err)
	}

	// 解析关键指标JSON
	if len(criticalMetricsJSON) > 0 {
		if err := json.Unmarshal(criticalMetricsJSON, &service.CriticalMetrics); err != nil {
			return nil, fmt.Errorf("解析服务关键指标失败: %w", err)
		}
	}

	return &service, nil
}

// GetByName 根据名称获取服务
func (r *PostgresServiceRepository) GetByName(ctx context.Context, name string) (*Service, error) {
	query := `
		SELECT 
			id, name, description, critical_metrics, created_time, updated_time
		FROM services
		WHERE name = $1
	`

	var service Service
	var criticalMetricsJSON []byte

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&service.ID,
		&service.Name,
		&service.Description,
		&criticalMetricsJSON,
		&service.CreatedAt,
		&service.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 未找到服务
		}
		return nil, fmt.Errorf("获取服务失败: %w", err)
	}

	// 解析关键指标JSON
	if len(criticalMetricsJSON) > 0 {
		if err := json.Unmarshal(criticalMetricsJSON, &service.CriticalMetrics); err != nil {
			return nil, fmt.Errorf("解析服务关键指标失败: %w", err)
		}
	}

	return &service, nil
}

// GetAll 获取所有服务
func (r *PostgresServiceRepository) GetAll(ctx context.Context) ([]*Service, error) {
	query := `
		SELECT 
			id, name, description, critical_metrics, created_time, updated_time
		FROM services
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询所有服务失败: %w", err)
	}
	defer rows.Close()

	return r.scanServices(rows)
}

// Update 更新服务
func (r *PostgresServiceRepository) Update(ctx context.Context, service *Service) error {
	// 将critical_metrics转换为JSONB
	criticalMetricsJSON, err := json.Marshal(service.CriticalMetrics)
	if err != nil {
		return fmt.Errorf("序列化服务关键指标失败: %w", err)
	}

	query := `
		UPDATE services
		SET 
			name = $2,
			description = $3,
			critical_metrics = $4
		WHERE id = $1
		RETURNING updated_time
	`

	err = r.db.QueryRowContext(
		ctx,
		query,
		service.ID,
		service.Name,
		service.Description,
		criticalMetricsJSON,
	).Scan(&service.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("服务 %s 不存在", service.ID)
		}
		return fmt.Errorf("更新服务失败: %w", err)
	}

	return nil
}

// Delete 删除服务
func (r *PostgresServiceRepository) Delete(ctx context.Context, id string) error {
	// 开始事务
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 首先更新引用此服务的节点
	updateNodesQuery := `
		UPDATE nodes
		SET service_id = NULL
		WHERE service_id = $1
	`

	_, err = tx.ExecContext(ctx, updateNodesQuery, id)
	if err != nil {
		return fmt.Errorf("更新引用服务的节点失败: %w", err)
	}

	// 然后删除服务
	deleteQuery := `DELETE FROM services WHERE id = $1`
	result, err := tx.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return fmt.Errorf("删除服务失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("服务 %s 不存在", id)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// scanServices 扫描查询结果并返回服务列表
func (r *PostgresServiceRepository) scanServices(rows *sql.Rows) ([]*Service, error) {
	var services []*Service

	for rows.Next() {
		var service Service
		var criticalMetricsJSON []byte

		err := rows.Scan(
			&service.ID,
			&service.Name,
			&service.Description,
			&criticalMetricsJSON,
			&service.CreatedAt,
			&service.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("扫描服务行失败: %w", err)
		}

		// 解析关键指标JSON
		if len(criticalMetricsJSON) > 0 {
			if err := json.Unmarshal(criticalMetricsJSON, &service.CriticalMetrics); err != nil {
				return nil, fmt.Errorf("解析服务关键指标失败: %w", err)
			}
		}

		services = append(services, &service)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("迭代服务行失败: %w", err)
	}

	return services, nil
}
