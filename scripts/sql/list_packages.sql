SELECT 
    owner || '/' || repo as package,
    version,
    CASE WHEN frozen = 1 THEN '✓' ELSE '' END as frozen,
    binary_name,
    datetime(installed_at) as installed,
    datetime(updated_at) as last_update
FROM packages
ORDER BY owner, repo; 