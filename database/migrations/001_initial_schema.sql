-- Initial schema for RPC Proxy Service
-- PostgreSQL database schema

CREATE TABLE IF NOT EXISTS rpc_endpoints (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    url VARCHAR(500) NOT NULL,
    weight INTEGER DEFAULT 1 CHECK (weight > 0),
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Health check history for analytics and monitoring
CREATE TABLE IF NOT EXISTS health_checks (
    id SERIAL PRIMARY KEY,
    endpoint_id INTEGER NOT NULL,
    healthy BOOLEAN NOT NULL,
    response_time_ms INTEGER,
    block_number VARCHAR(20),
    error_message TEXT,
    checked_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (endpoint_id) REFERENCES rpc_endpoints(id) ON DELETE CASCADE
);

-- System configuration settings
CREATE TABLE IF NOT EXISTS settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for better performance
CREATE INDEX IF NOT EXISTS idx_rpc_endpoints_enabled ON rpc_endpoints(enabled);
CREATE INDEX IF NOT EXISTS idx_health_checks_endpoint_id ON health_checks(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_health_checks_checked_at ON health_checks(checked_at);

-- Insert default settings
INSERT INTO settings (key, value, description) VALUES
    ('health_check_interval', '30s', 'Interval between health checks'),
    ('health_check_timeout', '5s', 'Timeout for each health check'),
    ('health_check_retries', '3', 'Number of retries before marking endpoint unhealthy'),
    ('proxy_timeout', '10s', 'Timeout for proxy requests'),
    ('max_connections', '1000', 'Maximum concurrent connections'),
    ('server_port', '8080', 'Server port number')
ON CONFLICT (key) DO NOTHING;

-- Insert default RPC endpoints
INSERT INTO rpc_endpoints (name, url, weight, enabled) VALUES
    ('LlamaRPC', 'https://eth.llamarpc.com', 1, true),
    ('PublicNode', 'https://ethereum.publicnode.com', 1, true),
    ('Cloudflare', 'https://cloudflare-eth.com', 1, true)
ON CONFLICT (name) DO NOTHING;