-- Add Soneium chain configuration
INSERT INTO chains (
    chain_id, 
    name, 
    display_name, 
    rpc_path, 
    is_testnet, 
    is_enabled,
    native_currency_symbol,
    native_currency_decimals,
    block_explorer_url
) VALUES (
    1868, 
    'soneium', 
    'Soneium Mainnet',
    'soneium',
    false,
    true,
    'ETH',
    18,
    'https://explorer.soneium.org'
);

-- Add Soneium RPC endpoints
INSERT INTO rpc_endpoints (chain_id, name, url, weight, enabled) VALUES
((SELECT id FROM chains WHERE name = 'soneium'), 'Soneium-DRPC', 'https://soneium.drpc.org', 3, true),
((SELECT id FROM chains WHERE name = 'soneium'), 'Soneium-Official', 'https://rpc.soneium.org', 2, true),
((SELECT id FROM chains WHERE name = 'soneium'), 'Soneium-PublicNode', 'https://soneium-mainnet.public.blastapi.io', 2, true);