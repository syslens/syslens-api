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

// NotificationStatus 定义通知状态类型
type NotificationStatus string

const (
	NotificationStatusTriggered    NotificationStatus = "triggered"
	NotificationStatusAcknowledged NotificationStatus = "acknowledged"
	NotificationStatusResolved     NotificationStatus = "resolved"
)

// Notification 表示通知实体
type Notification struct {
	ID          uuid.UUID          `json:"id"`
	AlertRuleID uuid.UUID          `json:"alert_rule_id"`
	NodeID      sql.NullString     `json:"node_id,omitempty"`
	TriggeredAt time.Time          `json:"triggered_at"`
	ResolvedAt  sql.NullTime       `json:"resolved_at,omitempty"`
	Status      NotificationStatus `json:"status"`
	Severity    AlertSeverity      `json:"severity"`
	Details     map[string]any     `json:"details,omitempty"`
}

// NotificationRepository 定义通知仓库接口
type NotificationRepository interface {
	// Create 创建新通知
	Create(ctx context.Context, notification *Notification) error

	// GetByID 根据ID获取通知
	GetByID(ctx context.Context, id uuid.UUID) (*Notification, error)

	// GetAll 获取所有通知
	GetAll(ctx context.Context, limit, offset int) ([]*Notification, error)

	// GetByAlertRuleID 获取特定告警规则的通知
	GetByAlertRuleID(ctx context.Context, alertRuleID uuid.UUID, limit, offset int) ([]*Notification, error)

	// GetByNodeID 获取特定节点的通知
	GetByNodeID(ctx context.Context, nodeID string, limit, offset int) ([]*Notification, error)

	// GetByStatus 获取特定状态的通知
	GetByStatus(ctx context.Context, status NotificationStatus, limit, offset int) ([]*Notification, error)

	// GetByTimeRange 获取特定时间范围内的通知
	GetByTimeRange(ctx context.Context, start, end time.Time, limit, offset int) ([]*Notification, error)

	// UpdateStatus 更新通知状态
	UpdateStatus(ctx context.Context, id uuid.UUID, status NotificationStatus) error

	// Resolve 解决通知
	Resolve(ctx context.Context, id uuid.UUID) error

	// Delete 删除通知
	Delete(ctx context.Context, id uuid.UUID) error

	// CleanupOldNotifications 清理旧通知
	CleanupOldNotifications(ctx context.Context, olderThan time.Time) (int, error)

	// Count 获取通知总数
	Count(ctx context.Context) (int, error)

	// CountByStatus 获取特定状态的通知数量
	CountByStatus(ctx context.Context, status NotificationStatus) (int, error)
}

// PostgresNotificationRepository 实现基于PostgreSQL的通知仓库
type PostgresNotificationRepository struct {
	db *storage.PostgresDB
}

// NewPostgresNotificationRepository 创建新的PostgreSQL通知仓库
func NewPostgresNotificationRepository(db *storage.PostgresDB) *PostgresNotificationRepository {
	return &PostgresNotificationRepository{
		db: db,
	}
}

// Create 创建新通知
func (r *PostgresNotificationRepository) Create(ctx context.Context, notification *Notification) error {
	// 将details转换为JSONB
	detailsJSON, err := json.Marshal(notification.Details)
	if err != nil {
		return fmt.Errorf("序列化通知详情失败: %w", err)
	}

	query := `
		INSERT INTO notifications (
			id, alert_rule_id, node_id, triggered_at, resolved_at, status, severity, details
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`

	// 如果ID为空，生成新的UUID
	if notification.ID == uuid.Nil {
		notification.ID = uuid.New()
	}

	_, err = r.db.ExecContext(
		ctx,
		query,
		notification.ID,
		notification.AlertRuleID,
		notification.NodeID,
		notification.TriggeredAt,
		notification.ResolvedAt,
		notification.Status,
		notification.Severity,
		detailsJSON,
	)

	if err != nil {
		return fmt.Errorf("创建通知失败: %w", err)
	}

	return nil
}

// GetByID 根据ID获取通知
func (r *PostgresNotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*Notification, error) {
	query := `
		SELECT 
			id, alert_rule_id, node_id, triggered_at, resolved_at, status, severity, details
		FROM notifications
		WHERE id = $1
	`

	var notification Notification
	var detailsJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&notification.ID,
		&notification.AlertRuleID,
		&notification.NodeID,
		&notification.TriggeredAt,
		&notification.ResolvedAt,
		&notification.Status,
		&notification.Severity,
		&detailsJSON,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 未找到通知
		}
		return nil, fmt.Errorf("获取通知失败: %w", err)
	}

	// 解析详情JSON
	if len(detailsJSON) > 0 {
		if err := json.Unmarshal(detailsJSON, &notification.Details); err != nil {
			return nil, fmt.Errorf("解析通知详情失败: %w", err)
		}
	}

	return &notification, nil
}

// GetAll 获取所有通知
func (r *PostgresNotificationRepository) GetAll(ctx context.Context, limit, offset int) ([]*Notification, error) {
	query := `
		SELECT 
			id, alert_rule_id, node_id, triggered_at, resolved_at, status, severity, details
		FROM notifications
		ORDER BY triggered_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("查询所有通知失败: %w", err)
	}
	defer rows.Close()

	return r.scanNotifications(rows)
}

// GetByAlertRuleID 获取特定告警规则的通知
func (r *PostgresNotificationRepository) GetByAlertRuleID(ctx context.Context, alertRuleID uuid.UUID, limit, offset int) ([]*Notification, error) {
	query := `
		SELECT 
			id, alert_rule_id, node_id, triggered_at, resolved_at, status, severity, details
		FROM notifications
		WHERE alert_rule_id = $1
		ORDER BY triggered_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, alertRuleID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("查询告警规则通知失败: %w", err)
	}
	defer rows.Close()

	return r.scanNotifications(rows)
}

// GetByNodeID 获取特定节点的通知
func (r *PostgresNotificationRepository) GetByNodeID(ctx context.Context, nodeID string, limit, offset int) ([]*Notification, error) {
	query := `
		SELECT 
			id, alert_rule_id, node_id, triggered_at, resolved_at, status, severity, details
		FROM notifications
		WHERE node_id = $1
		ORDER BY triggered_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, nodeID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("查询节点通知失败: %w", err)
	}
	defer rows.Close()

	return r.scanNotifications(rows)
}

// GetByStatus 获取特定状态的通知
func (r *PostgresNotificationRepository) GetByStatus(ctx context.Context, status NotificationStatus, limit, offset int) ([]*Notification, error) {
	query := `
		SELECT 
			id, alert_rule_id, node_id, triggered_at, resolved_at, status, severity, details
		FROM notifications
		WHERE status = $1
		ORDER BY triggered_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("查询状态通知失败: %w", err)
	}
	defer rows.Close()

	return r.scanNotifications(rows)
}

// GetByTimeRange 获取特定时间范围内的通知
func (r *PostgresNotificationRepository) GetByTimeRange(ctx context.Context, start, end time.Time, limit, offset int) ([]*Notification, error) {
	query := `
		SELECT 
			id, alert_rule_id, node_id, triggered_at, resolved_at, status, severity, details
		FROM notifications
		WHERE triggered_at BETWEEN $1 AND $2
		ORDER BY triggered_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, start, end, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("查询时间范围通知失败: %w", err)
	}
	defer rows.Close()

	return r.scanNotifications(rows)
}

// UpdateStatus 更新通知状态
func (r *PostgresNotificationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status NotificationStatus) error {
	query := `
		UPDATE notifications
		SET status = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("更新通知状态失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("通知 %s 不存在", id)
	}

	return nil
}

// Resolve 解决通知
func (r *PostgresNotificationRepository) Resolve(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE notifications
		SET 
			status = $2,
			resolved_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, NotificationStatusResolved, time.Now())
	if err != nil {
		return fmt.Errorf("解决通知失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("通知 %s 不存在", id)
	}

	return nil
}

// Delete 删除通知
func (r *PostgresNotificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notifications WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("删除通知失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("通知 %s 不存在", id)
	}

	return nil
}

// CleanupOldNotifications 清理旧通知
func (r *PostgresNotificationRepository) CleanupOldNotifications(ctx context.Context, olderThan time.Time) (int, error) {
	query := `DELETE FROM notifications WHERE triggered_at < $1`

	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("清理旧通知失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取受影响行数失败: %w", err)
	}

	return int(rowsAffected), nil
}

// Count 获取通知总数
func (r *PostgresNotificationRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM notifications`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("获取通知总数失败: %w", err)
	}

	return count, nil
}

// CountByStatus 获取特定状态的通知数量
func (r *PostgresNotificationRepository) CountByStatus(ctx context.Context, status NotificationStatus) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE status = $1`

	var count int
	err := r.db.QueryRowContext(ctx, query, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("获取状态通知数量失败: %w", err)
	}

	return count, nil
}

// scanNotifications 扫描查询结果并返回通知列表
func (r *PostgresNotificationRepository) scanNotifications(rows *sql.Rows) ([]*Notification, error) {
	var notifications []*Notification

	for rows.Next() {
		var notification Notification
		var detailsJSON []byte

		err := rows.Scan(
			&notification.ID,
			&notification.AlertRuleID,
			&notification.NodeID,
			&notification.TriggeredAt,
			&notification.ResolvedAt,
			&notification.Status,
			&notification.Severity,
			&detailsJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("扫描通知行失败: %w", err)
		}

		// 解析详情JSON
		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &notification.Details); err != nil {
				return nil, fmt.Errorf("解析通知详情失败: %w", err)
			}
		}

		notifications = append(notifications, &notification)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("迭代通知行失败: %w", err)
	}

	return notifications, nil
}
