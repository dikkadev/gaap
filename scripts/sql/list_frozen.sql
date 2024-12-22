SELECT 
    owner || '/' || repo as package,
    version,
    binary_name,
    datetime(updated_at) as last_update
FROM packages
WHERE frozen = 1
ORDER BY owner, repo; 