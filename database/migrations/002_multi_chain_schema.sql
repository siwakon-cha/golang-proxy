-- Multi-chain support schema
-- Add chains table for blockchain network metadata
CREATE TABLE IF NOT EXISTS chains (
    id SERIAL PRIMARY KEY,
    chain_id BIGINT NOT NULL UNIQUE,
    name VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    rpc_path VARCHAR(50) NOT NULL UNIQUE,
    is_testnet BOOLEAN DEFAULT false,
    native_currency_symbol VARCHAR(10) DEFAULT 'ETH',
    block_explorer_url VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    enabled BOOLEAN DEFAULT true
);

-- Add chain_config table for chain-specific configurations
CREATE TABLE IF NOT EXISTS chain_configs (
    id SERIAL PRIMARY KEY,
    chain_id INTEGER NOT NULL REFERENCES chains(id) ON DELETE CASCADE,
    config_key VARCHAR(100) NOT NULL,
    config_value TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(chain_id, config_key)
);

-- Add chain_id foreign key to existing rpc_endpoints table
ALTER TABLE rpc_endpoints 
ADD COLUMN IF NOT EXISTS chain_id INTEGER REFERENCES chains(id) ON DELETE CASCADE;

-- Create index on chain_id for better performance
CREATE INDEX IF NOT EXISTS idx_rpc_endpoints_chain_id ON rpc_endpoints(chain_id);
CREATE INDEX IF NOT EXISTS idx_rpc_endpoints_enabled_chain ON rpc_endpoints(enabled, chain_id);
CREATE INDEX IF NOT EXISTS idx_chains_enabled ON chains(enabled);
CREATE INDEX IF NOT EXISTS idx_chains_rpc_path ON chains(rpc_path);

-- Insert default chain configurations
INSERT INTO chains (chain_id, name, display_name, rpc_path, is_testnet, native_currency_symbol, block_explorer_url) VALUES
(1, 'ethereum', 'Ethereum Mainnet', 'ethereum', false, 'ETH', 'https://etherscan.io'),
(11155111, 'sepolia', 'Sepolia Testnet', 'sepolia', true, 'ETH', 'https://sepolia.etherscan.io'),
(1946, 'minato', 'Minato (Soneium Testnet)', 'minato', true, 'ETH', 'https://explorer-testnet.soneium.org'),
(1868, 'soneium', 'Soneium Mainnet', 'soneium', false, 'ETH', 'https://explorer.soneium.org')
ON CONFLICT (chain_id) DO NOTHING;

-- Insert default chain-specific configurations
INSERT INTO chain_configs (chain_id, config_key, config_value, description) VALUES
-- Ethereum Mainnet configs
((SELECT id FROM chains WHERE name = 'ethereum'), 'max_block_lag', '10', 'Maximum blocks behind latest considered healthy'),
((SELECT id FROM chains WHERE name = 'ethereum'), 'timeout_seconds', '15', 'Request timeout in seconds'),
((SELECT id FROM chains WHERE name = 'ethereum'), 'retry_attempts', '3', 'Number of retry attempts'),

-- Sepolia configs  
((SELECT id FROM chains WHERE name = 'sepolia'), 'max_block_lag', '20', 'Maximum blocks behind latest considered healthy'),
((SELECT id FROM chains WHERE name = 'sepolia'), 'timeout_seconds', '10', 'Request timeout in seconds'),
((SELECT id FROM chains WHERE name = 'sepolia'), 'retry_attempts', '3', 'Number of retry attempts'),

-- Minato configs
((SELECT id FROM chains WHERE name = 'minato'), 'max_block_lag', '50', 'Maximum blocks behind latest considered healthy'),
((SELECT id FROM chains WHERE name = 'minato'), 'timeout_seconds', '8', 'Request timeout in seconds'),
((SELECT id FROM chains WHERE name = 'minato'), 'retry_attempts', '2', 'Number of retry attempts'),

-- Soneium configs
((SELECT id FROM chains WHERE name = 'soneium'), 'max_block_lag', '30', 'Maximum blocks behind latest considered healthy'), 
((SELECT id FROM chains WHERE name = 'soneium'), 'timeout_seconds', '12', 'Request timeout in seconds'),
((SELECT id FROM chains WHERE name = 'soneium'), 'retry_attempts', '3', 'Number of retry attempts')
ON CONFLICT (chain_id, config_key) DO NOTHING;

-- Insert default RPC endpoints for each chain
INSERT INTO rpc_endpoints (name, url, weight, enabled, chain_id) VALUES
-- Ethereum Mainnet endpoints
('Ethereum-LlamaRPC', 'https://eth.llamarpc.com', 3, true, (SELECT id FROM chains WHERE name = 'ethereum')),
('Ethereum-PublicNode', 'https://ethereum.publicnode.com', 2, true, (SELECT id FROM chains WHERE name = 'ethereum')),
('Ethereum-Cloudflare', 'https://cloudflare-eth.com', 2, true, (SELECT id FROM chains WHERE name = 'ethereum')),

-- Sepolia endpoints
('Sepolia-1RPC', 'https://1rpc.io/sepolia', 3, true, (SELECT id FROM chains WHERE name = 'sepolia')),
('Sepolia-PublicNode', 'https://ethereum-sepolia-rpc.publicnode.com', 2, true, (SELECT id FROM chains WHERE name = 'sepolia')),
('Sepolia-DRPC', 'https://sepolia.drpc.org', 2, true, (SELECT id FROM chains WHERE name = 'sepolia')),

-- Minato (Soneium Testnet) endpoints
('Minato-Official', 'https://rpc.minato.soneium.org', 3, true, (SELECT id FROM chains WHERE name = 'minato')),

-- Soneium Mainnet endpoints (placeholder - update with actual endpoints)
('Soneium-Official', 'https://rpc.soneium.org', 3, true, (SELECT id FROM chains WHERE name = 'soneium'))
ON CONFLICT (name) DO NOTHING;

-- Update existing endpoints without chain_id to point to ethereum by default
UPDATE rpc_endpoints 
SET chain_id = (SELECT id FROM chains WHERE name = 'ethereum')
WHERE chain_id IS NULL;

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_chains_updated_at BEFORE UPDATE ON chains
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_chain_configs_updated_at BEFORE UPDATE ON chain_configs  
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();