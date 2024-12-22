SELECT 
    owner || '/' || repo as package,
    version,
    CASE WHEN frozen = 1 THEN '✓' ELSE '' END as frozen,
    binary_name,
    datetime(updated_at) as last_update,
    CAST((julianday('now') - julianday(updated_at)) AS INTEGER) as days_since_update
FROM packages
WHERE julianday('now') - julianday(updated_at) > 30
ORDER BY updated_at; 