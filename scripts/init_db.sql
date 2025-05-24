-- 启用TimescaleDB扩展
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    phone VARCHAR(20),
    nickname VARCHAR(50),
    avatar TEXT,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ
);

-- 股票基础信息表
CREATE TABLE IF NOT EXISTS stocks (
    symbol VARCHAR(20) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    market VARCHAR(10) NOT NULL,
    industry VARCHAR(50),
    sector VARCHAR(50),
    listing_date DATE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 股票行情表（时序表）
CREATE TABLE IF NOT EXISTS stock_quotes (
    id BIGSERIAL,
    symbol VARCHAR(20) NOT NULL,
    name VARCHAR(100),
    price DOUBLE PRECISION NOT NULL,
    open DOUBLE PRECISION,
    high DOUBLE PRECISION,
    low DOUBLE PRECISION,
    close DOUBLE PRECISION,
    pre_close DOUBLE PRECISION,
    volume DOUBLE PRECISION,
    amount DOUBLE PRECISION,
    turnover DOUBLE PRECISION,
    change DOUBLE PRECISION,
    change_percent DOUBLE PRECISION,
    timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 转换为超表
SELECT create_hypertable('stock_quotes', 'timestamp', if_not_exists => TRUE);

-- 用户订阅表
CREATE TABLE IF NOT EXISTS subscriptions (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    symbols JSONB NOT NULL,
    alert_rules JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_alert_at TIMESTAMPTZ
);

-- 异动事件表（时序表）
CREATE TABLE IF NOT EXISTS alert_events (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id VARCHAR(36) REFERENCES subscriptions(id) ON DELETE SET NULL,
    symbol VARCHAR(20) NOT NULL,
    stock_name VARCHAR(100),
    type VARCHAR(20) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    title VARCHAR(200) NOT NULL,
    message TEXT NOT NULL,
    quote_data JSONB,
    ai_analysis TEXT,
    intensity DOUBLE PRECISION,
    threshold DOUBLE PRECISION,
    is_read BOOLEAN DEFAULT false,
    is_notified BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 转换为超表
SELECT create_hypertable('alert_events', 'created_at', if_not_exists => TRUE);

-- 新闻事件表
CREATE TABLE IF NOT EXISTS news_events (
    id VARCHAR(36) PRIMARY KEY,
    symbol VARCHAR(20),
    title VARCHAR(500) NOT NULL,
    content TEXT,
    summary TEXT,
    source VARCHAR(100),
    author VARCHAR(100),
    url TEXT,
    sentiment VARCHAR(20) DEFAULT 'neutral',
    impact DOUBLE PRECISION DEFAULT 0.0,
    keywords JSONB,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 通知记录表
CREATE TABLE IF NOT EXISTS notification_records (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alert_id VARCHAR(36) REFERENCES alert_events(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    title VARCHAR(200),
    content TEXT,
    status VARCHAR(20) DEFAULT 'pending',
    sent_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 每日总结表
CREATE TABLE IF NOT EXISTS daily_summaries (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    alert_count INTEGER DEFAULT 0,
    top_symbols JSONB,
    summary TEXT,
    is_generated BOOLEAN DEFAULT false,
    is_sent BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, date)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_quotes_symbol_time ON stock_quotes (symbol, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_quotes_timestamp ON stock_quotes (timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON subscriptions (user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions (status);
CREATE INDEX IF NOT EXISTS idx_subscriptions_symbols ON subscriptions USING GIN (symbols);

CREATE INDEX IF NOT EXISTS idx_alerts_user_time ON alert_events (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_symbol_time ON alert_events (symbol, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_subscription ON alert_events (subscription_id);
CREATE INDEX IF NOT EXISTS idx_alerts_unread ON alert_events (user_id, is_read) WHERE is_read = false;

CREATE INDEX IF NOT EXISTS idx_news_symbol_time ON news_events (symbol, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_news_published ON news_events (published_at DESC);
CREATE INDEX IF NOT EXISTS idx_news_sentiment ON news_events (sentiment);

CREATE INDEX IF NOT EXISTS idx_notifications_user ON notification_records (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notification_records (status);

CREATE INDEX IF NOT EXISTS idx_summaries_user_date ON daily_summaries (user_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_summaries_pending ON daily_summaries (date) WHERE is_generated = false;