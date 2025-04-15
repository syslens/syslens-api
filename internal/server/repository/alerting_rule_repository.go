package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/syslens/syslens-api/internal/server/storage"
)

// AlertTargetType 定义告警目标类型
type AlertTargetType string

const (
	AlertTargetTypeNode    AlertTargetType = "node"
	AlertTargetTypeGroup   AlertTargetType = "group"
	AlertTargetTypeService AlertTargetType = "service"
	AlertTargetTypeGlobal  AlertTargetType = "global"
)

// AlertSeverity 定义告警严重级别
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertingRule 表示告警规则实体
type AlertingRule struct {
	ID                   uuid.UUID       `json:"id"`
	Name                 string          `json:"name"`
	Description          sql.NullString  `json:"description,omitempty"`
	TargetType           AlertTargetType `json:"target_type"`
	TargetID             sql.NullString  `json:"target_id,omitempty"`
	MetricQuery          string          `json:"metric_query"`
	Duration             time.Duration   `json:"duration"`
	Severity             AlertSeverity   `json:"severity"`
	NotificationChannels []string        `json:"notification_channels"`
	IsEnabled            bool            `json:"is_enabled"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

// AlertingRuleRepository 定义告警规则仓库接口
type AlertingRuleRepository interface {
	// Create 创建新告警规则
	Create(ctx context.Context, rule *AlertingRule) error

	// GetByID 根据ID获取告警规则
	GetByID(ctx context.Context, id uuid.UUID) (*AlertingRule, error)

	// GetAll 获取所有告警规则
	GetAll(ctx context.Context) ([]*AlertingRule, error)

	// GetByTarget 获取特定目标的告警规则
	GetByTarget(ctx context.Context, targetType AlertTargetType, targetID string) ([]*AlertingRule, error)

	// GetBySeverity 获取特定严重级别的告警规则
	GetBySeverity(ctx context.Context, severity AlertSeverity) ([]*AlertingRule, error)

	// GetEnabled 获取启用的告警规则
	GetEnabled(ctx context.Context) ([]*AlertingRule, error)

	// Update 更新告警规则
	Update(ctx context.Context, rule *AlertingRule) error

	// UpdateEnabled 启用/禁用告警规则
	UpdateEnabled(ctx context.Context, id uuid.UUID, isEnabled bool) error

	// Delete 删除告警规则
	Delete(ctx context.Context, id uuid.UUID) error
}

// PostgresAlertingRuleRepository 实现基于PostgreSQL的告警规则仓库
type PostgresAlertingRuleRepository struct {
	db *storage.PostgresDB
}

// NewPostgresAlertingRuleRepository 创建新的PostgreSQL告警规则仓库
func NewPostgresAlertingRuleRepository(db *storage.PostgresDB) *PostgresAlertingRuleRepository {
	return &PostgresAlertingRuleRepository{
		db: db,
	}
}

// Create 创建新告警规则
func (r *PostgresAlertingRuleRepository) Create(ctx context.Context, rule *AlertingRule) error {
	// 将通知渠道转换为JSONB
	channelsJSON, err := json.Marshal(rule.NotificationChannels)
	if err != nil {
		return fmt.Errorf("序列化通知渠道失败: %w", err)
	}

	query := `
		INSERT INTO alerting_rules (
			id, name, description, target_type, target_id, 
			metric_query, duration, severity, notification_channels, is_enabled
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		) RETURNING created_at, updated_at
	`

	// 如果ID为空，生成新的UUID
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}

	err = r.db.QueryRowContext(
		ctx,
		query,
		rule.ID,
		rule.Name,
		rule.Description,
		rule.TargetType,
		rule.TargetID,
		rule.MetricQuery,
		rule.Duration,
		rule.Severity,
		channelsJSON,
		rule.IsEnabled,
	).Scan(&rule.CreatedAt, &rule.UpdatedAt)

	if err != nil {
		return fmt.Errorf("创建告警规则失败: %w", err)
	}

	return nil
}

// GetByID 根据ID获取告警规则
func (r *PostgresAlertingRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*AlertingRule, error) {
	query := `
		SELECT 
			id, name, description, target_type, target_id, 
			metric_query, duration, severity, notification_channels, is_enabled,
			created_at, updated_at
		FROM alerting_rules
		WHERE id = $1
	`

	var rule AlertingRule
	var channelsJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rule.ID,
		&rule.Name,
		&rule.Description,
		&rule.TargetType,
		&rule.TargetID,
		&rule.MetricQuery,
		&rule.Duration,
		&rule.Severity,
		&channelsJSON,
		&rule.IsEnabled,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 未找到规则
		}
		return nil, fmt.Errorf("获取告警规则失败: %w", err)
	}

	// 解析通知渠道JSON
	if err := json.Unmarshal(channelsJSON, &rule.NotificationChannels); err != nil {
		return nil, fmt.Errorf("解析通知渠道失败: %w", err)
	}

	return &rule, nil
}

// GetAll 获取所有告警规则
func (r *PostgresAlertingRuleRepository) GetAll(ctx context.Context) ([]*AlertingRule, error) {
	query := `
		SELECT 
			id, name, description, target_type, target_id, 
			metric_query, duration, severity, notification_channels, is_enabled,
			created_at, updated_at
		FROM alerting_rules
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询所有告警规则失败: %w", err)
	}
	defer rows.Close()

	return r.scanRules(rows)
}

// GetByTarget 获取特定目标的告警规则
func (r *PostgresAlertingRuleRepository) GetByTarget(ctx context.Context, targetType AlertTargetType, targetID string) ([]*AlertingRule, error) {
	query := `
		SELECT 
			id, name, description, target_type, target_id, 
			metric_query, duration, severity, notification_channels, is_enabled,
			created_at, updated_at
		FROM alerting_rules
		WHERE target_type = $1 AND (target_id = $2 OR (target_type = 'global' AND target_id IS NULL))
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, targetType, targetID)
	if err != nil {
		return nil, fmt.Errorf("查询目标告警规则失败: %w", err)
	}
	defer rows.Close()

	return r.scanRules(rows)
}

// GetBySeverity 获取特定严重级别的告警规则
func (r *PostgresAlertingRuleRepository) GetBySeverity(ctx context.Context, severity AlertSeverity) ([]*AlertingRule, error) {
	query := `
		SELECT 
			id, name, description, target_type, target_id, 
			metric_query, duration, severity, notification_channels, is_enabled,
			created_at, updated_at
		FROM alerting_rules
		WHERE severity = $1
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, severity)
	if err != nil {
		return nil, fmt.Errorf("查询严重级别告警规则失败: %w", err)
	}
	defer rows.Close()

	return r.scanRules(rows)
}

// GetEnabled 获取启用的告警规则
func (r *PostgresAlertingRuleRepository) GetEnabled(ctx context.Context) ([]*AlertingRule, error) {
	query := `
		SELECT 
			id, name, description, target_type, target_id, 
			metric_query, duration, severity, notification_channels, is_enabled,
			created_at, updated_at
		FROM alerting_rules
		WHERE is_enabled = true
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询启用的告警规则失败: %w", err)
	}
	defer rows.Close()

	return r.scanRules(rows)
}

// Update 更新告警规则
func (r *PostgresAlertingRuleRepository) Update(ctx context.Context, rule *AlertingRule) error {
	// 将通知渠道转换为JSONB
	channelsJSON, err := json.Marshal(rule.NotificationChannels)
	if err != nil {
		return fmt.Errorf("序列化通知渠道失败: %w", err)
	}

	query := `
		UPDATE alerting_rules
		SET 
			name = $2,
			description = $3,
			target_type = $4,
			target_id = $5,
			metric_query = $6,
			duration = $7,
			severity = $8,
			notification_channels = $9,
			is_enabled = $10
		WHERE id = $1
		RETURNING updated_at
	`

	err = r.db.QueryRowContext(
		ctx,
		query,
		rule.ID,
		rule.Name,
		rule.Description,
		rule.TargetType,
		rule.TargetID,
		rule.MetricQuery,
		rule.Duration,
		rule.Severity,
		channelsJSON,
		rule.IsEnabled,
	).Scan(&rule.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("告警规则 %s 不存在", rule.ID)
		}
		return fmt.Errorf("更新告警规则失败: %w", err)
	}

	return nil
}

// UpdateEnabled 启用/禁用告警规则
func (r *PostgresAlertingRuleRepository) UpdateEnabled(ctx context.Context, id uuid.UUID, isEnabled bool) error {
	query := `
		UPDATE alerting_rules
		SET is_enabled = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, isEnabled)
	if err != nil {
		return fmt.Errorf("更新告警规则状态失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("告警规则 %s 不存在", id)
	}

	return nil
}

// Delete 删除告警规则
func (r *PostgresAlertingRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// 开始事务
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 删除相关的通知记录
	deleteNotificationsQuery := `DELETE FROM notifications WHERE alert_rule_id = $1`
	_, err = tx.ExecContext(ctx, deleteNotificationsQuery, id)
	if err != nil {
		return fmt.Errorf("删除告警规则相关通知失败: %w", err)
	}

	// 删除告警规则
	deleteRuleQuery := `DELETE FROM alerting_rules WHERE id = $1`
	result, err := tx.ExecContext(ctx, deleteRuleQuery, id)
	if err != nil {
		return fmt.Errorf("删除告警规则失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("告警规则 %s 不存在", id)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// scanRules 扫描查询结果并返回告警规则列表
func (r *PostgresAlertingRuleRepository) scanRules(rows *sql.Rows) ([]*AlertingRule, error) {
	var rules []*AlertingRule

	for rows.Next() {
		var rule AlertingRule
		var channelsJSON []byte

		err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Description,
			&rule.TargetType,
			&rule.TargetID,
			&rule.MetricQuery,
			&rule.Duration,
			&rule.Severity,
			&channelsJSON,
			&rule.IsEnabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("扫描告警规则行失败: %w", err)
		}

		// 解析通知渠道JSON
		if err := json.Unmarshal(channelsJSON, &rule.NotificationChannels); err != nil {
			return nil, fmt.Errorf("解析通知渠道失败: %w", err)
		}

		rules = append(rules, &rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("迭代告警规则行失败: %w", err)
	}

	return rules, nil
}
