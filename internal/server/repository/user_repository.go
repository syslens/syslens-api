package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/syslens/syslens-api/internal/server/storage"
)

// UserRole 定义用户角色类型
type UserRole string

const (
	UserRoleAdmin  UserRole = "admin"
	UserRoleEditor UserRole = "editor"
	UserRoleViewer UserRole = "viewer"
)

// User 表示用户实体
type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // 不在JSON中暴露
	Role         UserRole  `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserSession 表示用户会话实体
type UserSession struct {
	SessionID  string    `json:"session_id"`
	UserID     uuid.UUID `json:"user_id"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
}

// UserRepository 定义用户仓库接口
type UserRepository interface {
	// Create 创建新用户
	Create(ctx context.Context, user *User) error

	// GetByID 根据ID获取用户
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)

	// GetByUsername 根据用户名获取用户
	GetByUsername(ctx context.Context, username string) (*User, error)

	// GetByEmail 根据邮箱获取用户
	GetByEmail(ctx context.Context, email string) (*User, error)

	// GetAll 获取所有用户
	GetAll(ctx context.Context) ([]*User, error)

	// Update 更新用户
	Update(ctx context.Context, user *User) error

	// UpdatePassword 更新用户密码
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error

	// Delete 删除用户
	Delete(ctx context.Context, id uuid.UUID) error

	// CreateSession 创建用户会话
	CreateSession(ctx context.Context, session *UserSession) error

	// GetSession 获取会话
	GetSession(ctx context.Context, sessionID string) (*UserSession, error)

	// UpdateSessionLastUsed 更新会话最后使用时间
	UpdateSessionLastUsed(ctx context.Context, sessionID string) error

	// DeleteSession 删除会话
	DeleteSession(ctx context.Context, sessionID string) error

	// CleanupExpiredSessions 清理过期会话
	CleanupExpiredSessions(ctx context.Context) (int, error)
}

// PostgresUserRepository 实现基于PostgreSQL的用户仓库
type PostgresUserRepository struct {
	db *storage.PostgresDB
}

// NewPostgresUserRepository 创建新的PostgreSQL用户仓库
func NewPostgresUserRepository(db *storage.PostgresDB) *PostgresUserRepository {
	return &PostgresUserRepository{
		db: db,
	}
}

// Create 创建新用户
func (r *PostgresUserRepository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (
			id, username, email, password_hash, role, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6
		) RETURNING created_at, updated_at
	`

	// 如果ID为空，生成新的UUID
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	err := r.db.QueryRowContext(
		ctx,
		query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.IsActive,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}

	return nil
}

// GetByID 根据ID获取用户
func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT 
			id, username, email, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 未找到用户
		}
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (r *PostgresUserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	query := `
		SELECT 
			id, username, email, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 未找到用户
		}
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	return &user, nil
}

// GetByEmail 根据邮箱获取用户
func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT 
			id, username, email, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 未找到用户
		}
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	return &user, nil
}

// GetAll 获取所有用户
func (r *PostgresUserRepository) GetAll(ctx context.Context) ([]*User, error) {
	query := `
		SELECT 
			id, username, email, password_hash, role, is_active, created_at, updated_at
		FROM users
		ORDER BY username
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询所有用户失败: %w", err)
	}
	defer rows.Close()

	return r.scanUsers(rows)
}

// Update 更新用户
func (r *PostgresUserRepository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE users
		SET 
			username = $2,
			email = $3,
			role = $4,
			is_active = $5
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		user.ID,
		user.Username,
		user.Email,
		user.Role,
		user.IsActive,
	).Scan(&user.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("用户 %s 不存在", user.ID)
		}
		return fmt.Errorf("更新用户失败: %w", err)
	}

	return nil
}

// UpdatePassword 更新用户密码
func (r *PostgresUserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, passwordHash)
	if err != nil {
		return fmt.Errorf("更新用户密码失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("用户 %s 不存在", id)
	}

	return nil
}

// Delete 删除用户
func (r *PostgresUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// 开始事务
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 首先删除用户的所有会话
	deleteSessionsQuery := `DELETE FROM user_sessions WHERE user_id = $1`
	_, err = tx.ExecContext(ctx, deleteSessionsQuery, id)
	if err != nil {
		return fmt.Errorf("删除用户会话失败: %w", err)
	}

	// 然后删除用户
	deleteUserQuery := `DELETE FROM users WHERE id = $1`
	result, err := tx.ExecContext(ctx, deleteUserQuery, id)
	if err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("用户 %s 不存在", id)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// CreateSession 创建用户会话
func (r *PostgresUserRepository) CreateSession(ctx context.Context, session *UserSession) error {
	query := `
		INSERT INTO user_sessions (
			session_id, user_id, expires_at
		) VALUES (
			$1, $2, $3
		) RETURNING created_at, last_used_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		session.SessionID,
		session.UserID,
		session.ExpiresAt,
	).Scan(&session.CreatedAt, &session.LastUsedAt)

	if err != nil {
		return fmt.Errorf("创建用户会话失败: %w", err)
	}

	return nil
}

// GetSession 获取会话
func (r *PostgresUserRepository) GetSession(ctx context.Context, sessionID string) (*UserSession, error) {
	query := `
		SELECT 
			session_id, user_id, expires_at, created_at, last_used_at
		FROM user_sessions
		WHERE session_id = $1
	`

	var session UserSession
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.SessionID,
		&session.UserID,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.LastUsedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 未找到会话
		}
		return nil, fmt.Errorf("获取用户会话失败: %w", err)
	}

	return &session, nil
}

// UpdateSessionLastUsed 更新会话最后使用时间
func (r *PostgresUserRepository) UpdateSessionLastUsed(ctx context.Context, sessionID string) error {
	query := `
		UPDATE user_sessions
		SET last_used_at = NOW()
		WHERE session_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("更新会话最后使用时间失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("会话 %s 不存在", sessionID)
	}

	return nil
}

// DeleteSession 删除会话
func (r *PostgresUserRepository) DeleteSession(ctx context.Context, sessionID string) error {
	query := `DELETE FROM user_sessions WHERE session_id = $1`

	result, err := r.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("删除会话失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取受影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("会话 %s 不存在", sessionID)
	}

	return nil
}

// CleanupExpiredSessions 清理过期会话
func (r *PostgresUserRepository) CleanupExpiredSessions(ctx context.Context) (int, error) {
	query := `DELETE FROM user_sessions WHERE expires_at < NOW()`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("清理过期会话失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取受影响行数失败: %w", err)
	}

	return int(rowsAffected), nil
}

// scanUsers 扫描查询结果并返回用户列表
func (r *PostgresUserRepository) scanUsers(rows *sql.Rows) ([]*User, error) {
	var users []*User

	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("扫描用户行失败: %w", err)
		}

		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("迭代用户行失败: %w", err)
	}

	return users, nil
}
