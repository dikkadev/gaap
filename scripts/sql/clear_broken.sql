-- First, show what will be deleted
SELECT 
    owner || '/' || repo as package,
    version,
    binary_name,
    install_path,
    CASE 
        WHEN install_path NOT LIKE '%/bin/actual/%' THEN 'Invalid path (not in bin/actual)'
        WHEN install_path NOT LIKE '%-%-%' THEN 'Invalid path format'
        WHEN binary_name = '' OR trim(binary_name) = '' THEN 'Empty or whitespace binary name'
        ELSE 'Invalid package'
    END as reason
FROM packages
WHERE install_path NOT LIKE '%/bin/actual/%'  -- Check path format
   OR install_path NOT LIKE '%-%-%'  -- Check expected naming pattern
   OR binary_name = ''  -- Empty string
   OR trim(binary_name) = ''  -- Whitespace only
ORDER BY owner, repo; 