-- SysLens Control Plane Initial Schema (PostgreSQL)

-- 启用 UUID 生成功能 (如果尚未启用)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 自动更新 updated_at 时间戳的函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = NOW();
   RETURN NEW;
END;
$$ language 'plpgsql';

-- 1. 节点信息表
CREATE TABLE nodes (
    id VARCHAR(255) PRIMARY KEY,             -- 节点唯一ID
    name VARCHAR(255) NOT NULL,            -- 节点名称
    auth_token_hash VARCHAR(255) NOT NULL, -- 节点的认证令牌哈希值
    labels JSONB DEFAULT '{}'::jsonb,       -- 节点标签
    type VARCHAR(50) DEFAULT 'agent',      -- 节点类型 ('agent', 'fixed-service')
    status VARCHAR(50) DEFAULT 'pending',   -- 节点状态 ('pending', 'active', 'inactive')
    group_id VARCHAR(255),                 -- 所属分组ID (外键)
    service_id VARCHAR(255),               -- 关联的服务ID (外键)
    description TEXT,                      -- 描述信息
    registered_at TIMESTAMPTZ,             -- 首次注册时间
    last_active_at TIMESTAMPTZ,            -- 最后活跃时间
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 索引和触发器 for nodes
CREATE INDEX idx_nodes_name ON nodes (name);
CREATE INDEX idx_nodes_status ON nodes (status);
CREATE INDEX idx_nodes_last_active_at ON nodes (last_active_at);
CREATE INDEX idx_nodes_labels ON nodes USING GIN (labels);
CREATE INDEX idx_nodes_group_id ON nodes (group_id);
CREATE INDEX idx_nodes_service_id ON nodes (service_id);

CREATE TRIGGER update_nodes_updated_at
BEFORE UPDATE ON nodes
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 2. 节点分组表
CREATE TABLE node_groups (
    id VARCHAR(255) PRIMARY KEY,             -- 分组唯一ID
    name VARCHAR(255) NOT NULL UNIQUE,       -- 分组名称
    type VARCHAR(100),                     -- 分组类型 (例如: 'region', 'function', 'environment')
    description TEXT,                      -- 描述
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_node_groups_updated_at
BEFORE UPDATE ON node_groups
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 3. 固定服务表
CREATE TABLE services (
    id VARCHAR(255) PRIMARY KEY,             -- 服务唯一ID
    name VARCHAR(255) NOT NULL UNIQUE,       -- 服务名称
    description TEXT,                      -- 描述
    critical_metrics JSONB DEFAULT '{}'::jsonb, -- 服务的关键指标定义 (可选)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_services_updated_at
BEFORE UPDATE ON services
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 添加外键约束 (在表创建后)
ALTER TABLE nodes ADD CONSTRAINT fk_nodes_group FOREIGN KEY (group_id) REFERENCES node_groups(id) ON DELETE SET NULL;
ALTER TABLE nodes ADD CONSTRAINT fk_nodes_service FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE SET NULL;

-- 4. 用户表
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), -- 用户唯一ID
    username VARCHAR(100) NOT NULL UNIQUE,          -- 用户名
    email VARCHAR(255) NOT NULL UNIQUE,             -- 邮箱
    password_hash VARCHAR(255) NOT NULL,          -- 哈希后的密码 (bcrypt/Argon2)
    role VARCHAR(50) NOT NULL DEFAULT 'viewer',     -- 用户角色 ('admin', 'editor', 'viewer')
    is_active BOOLEAN NOT NULL DEFAULT true,      -- 账号是否激活
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_role ON users (role);

CREATE TRIGGER update_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 5. 用户会话/令牌表 (示例，具体实现可能不同，如使用JWT则不需要此表)
CREATE TABLE user_sessions (
    session_id VARCHAR(255) PRIMARY KEY,           -- 会话ID (安全随机字符串)
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- 关联用户ID
    expires_at TIMESTAMPTZ NOT NULL,                -- 过期时间
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- 创建时间
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT NOW() -- 最后使用时间
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions (user_id);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions (expires_at);

-- 6. 告警规则表
CREATE TABLE alerting_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), -- 规则唯一ID
    name VARCHAR(255) NOT NULL,                   -- 规则名称
    description TEXT,                           -- 规则描述
    target_type VARCHAR(50) NOT NULL,             -- 目标类型 ('node', 'group', 'service', 'global')
    target_id VARCHAR(255),                     -- 目标ID (node_id, group_id, service_id, or NULL for global)
    metric_query TEXT NOT NULL,                   -- 告警条件 (例如: 'cpu.usage > 90')
    duration INTERVAL NOT NULL,                   -- 持续时间 (例如: '5 minutes')
    severity VARCHAR(50) NOT NULL DEFAULT 'warning', -- 严重级别 ('info', 'warning', 'critical')
    notification_channels JSONB DEFAULT '[]'::jsonb, -- 通知渠道 (例如: ["email", "webhook-slack"])
    is_enabled BOOLEAN NOT NULL DEFAULT true,     -- 是否启用
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alerting_rules_target ON alerting_rules (target_type, target_id);
CREATE INDEX idx_alerting_rules_severity ON alerting_rules (severity);
CREATE INDEX idx_alerting_rules_is_enabled ON alerting_rules (is_enabled);

CREATE TRIGGER update_alerting_rules_updated_at
BEFORE UPDATE ON alerting_rules
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 7. (可选) 通知历史记录表
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    alert_rule_id UUID NOT NULL REFERENCES alerting_rules(id) ON DELETE CASCADE,
    node_id VARCHAR(255), -- 触发告警的节点 (可能为空)
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    status VARCHAR(50) NOT NULL, -- (e.g., 'triggered', 'acknowledged', 'resolved')
    severity VARCHAR(50) NOT NULL,
    details JSONB -- 触发时的具体指标值等
);
CREATE INDEX idx_notifications_alert_rule_id ON notifications (alert_rule_id);
CREATE INDEX idx_notifications_status ON notifications (status);
CREATE INDEX idx_notifications_triggered_at ON notifications (triggered_at);

-- 可以在这里添加其他初始化表，例如全局 settings 等

-- migration: 001_create_initial_tables.sql ends here 