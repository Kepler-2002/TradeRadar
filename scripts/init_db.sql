-- 创建扩展
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- 创建股票行情表
CREATE TABLE IF NOT EXISTS stock_quotes (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    volume DOUBLE PRECISION NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL
);

-- 将表转换为超表
SELECT create_hypertable('stock_quotes', 'timestamp', if_not_exists => TRUE);

-- 创建股票异动表
CREATE TABLE IF NOT EXISTS stock_alerts (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    alert_type INTEGER NOT NULL,
    intensity DOUBLE PRECISION NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL
);

-- 将表转换为超表
SELECT create_hypertable('stock_alerts', 'timestamp', if_not_exists => TRUE);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_quotes_symbol ON stock_quotes (symbol, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_symbol ON stock_alerts (symbol, timestamp DESC);

-- 创建用户订阅表
CREATE TABLE IF NOT EXISTS user_subscriptions (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    rule_type INTEGER NOT NULL,
    threshold DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON user_subscriptions (user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_symbol ON user_subscriptions (symbol);