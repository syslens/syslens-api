package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq" // PostgreSQL驱动
)

// PostgresConfig 保存PostgreSQL数据库连接配置
type PostgresConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	DBName       string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	ConnMaxLife  time.Duration // 连接最大生命周期，time.Duration类型
	AutoMigrate  bool
}

// PostgresDB 提供PostgreSQL数据库连接和操作
type PostgresDB struct {
	db     *sql.DB
	config PostgresConfig
}

// NewPostgresDB 创建新的PostgreSQL数据库连接
func NewPostgresDB(config PostgresConfig) (*PostgresDB, error) {
	// 构建连接字符串
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
	)

	// 打开数据库连接
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("打开PostgreSQL连接失败: %w", err)
	}

	// 配置连接池
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLife > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLife)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("PostgreSQL连接测试失败: %w", err)
	}

	log.Printf("成功连接到PostgreSQL数据库: %s:%d/%s", config.Host, config.Port, config.DBName)

	return &PostgresDB{
		db:     db,
		config: config,
	}, nil
}

// GetDB 返回底层数据库连接
func (p *PostgresDB) GetDB() *sql.DB {
	return p.db
}

// Close 关闭数据库连接
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// ExecContext 执行SQL语句
func (p *PostgresDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return p.db.ExecContext(ctx, query, args...)
}

// QueryContext 执行查询SQL语句
func (p *PostgresDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, query, args...)
}

// QueryRowContext 执行查询SQL语句并返回单行结果
func (p *PostgresDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return p.db.QueryRowContext(ctx, query, args...)
}

// BeginTx 开始事务
func (p *PostgresDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return p.db.BeginTx(ctx, opts)
}

// PrepareContext 准备SQL语句
func (p *PostgresDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return p.db.PrepareContext(ctx, query)
}
