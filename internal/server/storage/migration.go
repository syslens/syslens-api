package storage

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

// 数据库表定义
const (
	createUsersTable = `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY,
		username VARCHAR(255) NOT NULL UNIQUE,
		email VARCHAR(255) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);
	`

	createUserSessionsTable = `
	CREATE TABLE IF NOT EXISTS user_sessions (
		session_id VARCHAR(255) PRIMARY KEY,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		last_used_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);
	`

	createGroupsTable = `
	CREATE TABLE IF NOT EXISTS node_groups (
		id UUID PRIMARY KEY,
		name VARCHAR(255) NOT NULL UNIQUE,
		description TEXT,
		type VARCHAR(50),
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);
	`

	createNodesTable = `
	CREATE TABLE IF NOT EXISTS nodes (
		id VARCHAR(255) PRIMARY KEY,
		hostname VARCHAR(255),
		ip_address VARCHAR(50),
		status VARCHAR(50) NOT NULL,
		group_id UUID REFERENCES node_groups(id) ON DELETE SET NULL,
		service_id VARCHAR(255) REFERENCES services(id) ON DELETE SET NULL,
		labels JSONB,
		last_active TIMESTAMP WITH TIME ZONE,
		registered_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);
	`

	createServicesTable = `
	CREATE TABLE IF NOT EXISTS services (
		id VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) NOT NULL UNIQUE,
		description TEXT,
		type VARCHAR(50),
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);
	`

	createServiceNodesTable = `
	CREATE TABLE IF NOT EXISTS service_nodes (
		service_id VARCHAR(255) NOT NULL REFERENCES services(id) ON DELETE CASCADE,
		node_id VARCHAR(255) NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
		priority INT NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		PRIMARY KEY (service_id, node_id)
	);
	`

	createAlertRulesTable = `
	CREATE TABLE IF NOT EXISTS alert_rules (
		id UUID PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		query TEXT NOT NULL,
		severity VARCHAR(50) NOT NULL,
		threshold FLOAT NOT NULL,
		duration INT NOT NULL,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);
	`

	createNotificationsTable = `
	CREATE TABLE IF NOT EXISTS notifications (
		id UUID PRIMARY KEY,
		alert_rule_id UUID REFERENCES alert_rules(id) ON DELETE SET NULL,
		node_id VARCHAR(255) REFERENCES nodes(id) ON DELETE SET NULL,
		title VARCHAR(255) NOT NULL,
		message TEXT NOT NULL,
		severity VARCHAR(50) NOT NULL,
		status VARCHAR(50) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		resolved_at TIMESTAMP WITH TIME ZONE
	);
	`
)

// 数据库迁移列表
var migrationSchemas = []string{
	createServicesTable, // 先创建服务表，因为节点表依赖它
	createGroupsTable,   // 先创建分组表，因为节点表依赖它
	createNodesTable,    // 节点表依赖服务表和分组表
	createServiceNodesTable,
	createUsersTable,
	createUserSessionsTable,
	createAlertRulesTable,
	createNotificationsTable,
}

// MigrateDatabase 执行数据库迁移
func (p *PostgresDB) MigrateDatabase(ctx context.Context) error {
	log.Println("开始数据库迁移...")

	// 首先检查连接是否正常
	if err := p.db.PingContext(ctx); err != nil {
		return fmt.Errorf("数据库连接检查失败: %w", err)
	}

	// 创建迁移表
	if err := p.createMigrationTable(ctx); err != nil {
		return err
	}

	// 确认迁移表是否已经创建成功
	exists, err := p.tableExists(ctx, "schema_migrations")
	if err != nil {
		return fmt.Errorf("检查迁移表是否存在失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("创建迁移表失败，请检查数据库权限")
	}

	// 开始执行迁移
	for i, schema := range migrationSchemas {
		migrationName := fmt.Sprintf("migration_%d", i+1)

		// 检查是否已经执行过
		if migrated, err := p.isMigrationApplied(ctx, migrationName); err != nil {
			return err
		} else if migrated {
			log.Printf("迁移 '%s' 已经应用，跳过", migrationName)
			continue
		}

		// 应用迁移
		log.Printf("应用迁移 '%s'...", migrationName)
		if _, err := p.db.ExecContext(ctx, schema); err != nil {
			return fmt.Errorf("执行迁移 '%s' 失败: %w", migrationName, err)
		}

		// 记录迁移
		if err := p.recordMigration(ctx, migrationName); err != nil {
			return err
		}

		log.Printf("迁移 '%s' 应用成功", migrationName)
	}

	log.Println("数据库迁移完成")
	return nil
}

// CheckTablesExist 检查所有必需的表是否存在
func (p *PostgresDB) CheckTablesExist(ctx context.Context) error {
	requiredTables := []string{
		"users", "user_sessions", "node_groups", "nodes",
		"services", "service_nodes", "alert_rules", "notifications",
	}

	log.Println("检查数据库表结构...")

	for _, table := range requiredTables {
		exists, err := p.tableExists(ctx, table)
		if err != nil {
			return fmt.Errorf("检查表 '%s' 失败: %w", table, err)
		}

		if !exists {
			return fmt.Errorf("必需的表 '%s' 不存在，请执行数据库迁移", table)
		}
	}

	log.Println("所有必需的数据库表都已存在")
	return nil
}

// VerifyTableColumns 验证表的列是否符合预期
func (p *PostgresDB) VerifyTableColumns(ctx context.Context) error {
	type tableColumns struct {
		tableName string
		columns   []string
	}

	// 定义每个表必需的列
	requiredColumns := []tableColumns{
		{
			tableName: "users",
			columns:   []string{"id", "username", "email", "password_hash", "role", "is_active", "created_at", "updated_at"},
		},
		{
			tableName: "user_sessions",
			columns:   []string{"session_id", "user_id", "expires_at", "created_at", "last_used_at"},
		},
		{
			tableName: "node_groups",
			columns:   []string{"id", "name", "description", "type", "created_at", "updated_at"},
		},
		{
			tableName: "nodes",
			columns:   []string{"id", "hostname", "ip_address", "status", "group_id", "service_id", "labels", "last_active", "registered_at", "updated_at"},
		},
		{
			tableName: "services",
			columns:   []string{"id", "name", "description", "type", "created_at", "updated_at"},
		},
		{
			tableName: "service_nodes",
			columns:   []string{"service_id", "node_id", "priority", "created_at"},
		},
		{
			tableName: "alert_rules",
			columns:   []string{"id", "name", "description", "query", "severity", "threshold", "duration", "is_active", "created_at", "updated_at"},
		},
		{
			tableName: "notifications",
			columns:   []string{"id", "alert_rule_id", "node_id", "title", "message", "severity", "status", "created_at", "updated_at", "resolved_at"},
		},
	}

	log.Println("验证表列结构...")

	// 检查每个表的列
	for _, tc := range requiredColumns {
		columns, err := p.getTableColumns(ctx, tc.tableName)
		if err != nil {
			return fmt.Errorf("获取表 '%s' 的列失败: %w", tc.tableName, err)
		}

		// 检查是否所有必需的列都存在
		for _, requiredCol := range tc.columns {
			found := false
			for _, col := range columns {
				if strings.EqualFold(col, requiredCol) {
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("表 '%s' 缺少必需的列 '%s'", tc.tableName, requiredCol)
			}
		}
	}

	log.Println("所有表的列结构验证通过")
	return nil
}

// createMigrationTable 创建迁移记录表
func (p *PostgresDB) createMigrationTable(ctx context.Context) error {
	// 首先检查表是否已存在
	exists, err := p.tableExists(ctx, "schema_migrations")
	if err != nil {
		return fmt.Errorf("检查迁移表是否存在失败: %w", err)
	}

	if exists {
		log.Println("迁移表已存在，跳过创建")
		return nil
	}

	log.Println("创建迁移表...")
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		migration_name VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);
	`

	_, err = p.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("创建迁移表失败: %w", err)
	}

	log.Println("迁移表创建成功")
	return nil
}

// isMigrationApplied 检查迁移是否已经应用
func (p *PostgresDB) isMigrationApplied(ctx context.Context, migrationName string) (bool, error) {
	// 首先检查表是否存在
	exists, err := p.tableExists(ctx, "schema_migrations")
	if err != nil {
		return false, fmt.Errorf("检查迁移表是否存在失败: %w", err)
	}
	if !exists {
		// 表不存在，意味着没有迁移被应用过
		return false, nil
	}

	query := `
	SELECT EXISTS (
		SELECT 1 FROM schema_migrations WHERE migration_name = $1
	);
	`

	var migrationExists bool
	err = p.db.QueryRowContext(ctx, query, migrationName).Scan(&migrationExists)
	if err != nil {
		return false, fmt.Errorf("检查迁移状态失败: %w", err)
	}

	return migrationExists, nil
}

// recordMigration 记录迁移
func (p *PostgresDB) recordMigration(ctx context.Context, migrationName string) error {
	query := `
	INSERT INTO schema_migrations (migration_name, applied_at)
	VALUES ($1, $2);
	`

	_, err := p.db.ExecContext(ctx, query, migrationName, time.Now())
	if err != nil {
		return fmt.Errorf("记录迁移状态失败: %w", err)
	}

	return nil
}

// tableExists 检查表是否存在
func (p *PostgresDB) tableExists(ctx context.Context, tableName string) (bool, error) {
	query := `
	SELECT EXISTS (
		SELECT FROM pg_tables
		WHERE schemaname = 'public' AND tablename = $1
	);
	`

	var exists bool
	err := p.db.QueryRowContext(ctx, query, tableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("检查表是否存在失败: %w", err)
	}

	return exists, nil
}

// getTableColumns 获取表的所有列名
func (p *PostgresDB) getTableColumns(ctx context.Context, tableName string) ([]string, error) {
	query := `
	SELECT column_name
	FROM information_schema.columns
	WHERE table_schema = 'public' AND table_name = $1
	ORDER BY ordinal_position;
	`

	rows, err := p.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("查询表列信息失败: %w", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, err
		}
		columns = append(columns, column)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}

// CheckDatabaseHealth 检查数据库健康状态
func (p *PostgresDB) CheckDatabaseHealth(ctx context.Context) error {
	// 设置较短的超时时间以便快速检测
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// 尝试 ping 数据库
	if err := p.db.PingContext(timeoutCtx); err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 尝试执行简单查询
	var one int
	err := p.db.QueryRowContext(timeoutCtx, "SELECT 1").Scan(&one)
	if err != nil {
		return fmt.Errorf("数据库查询测试失败: %w", err)
	}

	if one != 1 {
		return errors.New("数据库查询返回意外结果")
	}

	// 检查连接池状态
	stats := p.db.Stats()
	log.Printf("数据库连接池状态: 打开=%d, 使用中=%d, 空闲=%d",
		stats.OpenConnections, stats.InUse, stats.Idle)

	// 如果连接池接近耗尽，记录警告
	if stats.OpenConnections > 0 && float64(stats.InUse)/float64(stats.OpenConnections) > 0.8 {
		log.Printf("警告: 数据库连接池使用率高 (%.2f%%)",
			float64(stats.InUse)/float64(stats.OpenConnections)*100)
	}

	return nil
}
