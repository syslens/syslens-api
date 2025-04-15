-- Down Migration for 001_create_initial_tables

-- 删除外键约束
ALTER TABLE nodes DROP CONSTRAINT IF EXISTS fk_nodes_group;
ALTER TABLE nodes DROP CONSTRAINT IF EXISTS fk_nodes_service;

-- 删除表 (按依赖相反顺序)
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS alerting_rules;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS node_groups;
DROP TABLE IF EXISTS nodes;

-- 删除自动更新 updated_at 的函数 (如果不再被其他表使用)
-- 注意: 如果有其他迁移脚本也使用了这个函数，不应在此处删除
DROP FUNCTION IF EXISTS update_updated_at_column();

-- 删除 uuid-ossp 扩展 (如果不再需要)
-- 注意: 如果有其他表使用了UUID，不应在此处删除
-- DROP EXTENSION IF EXISTS "uuid-ossp";

-- migration: 001_create_initial_tables.down.sql ends here 