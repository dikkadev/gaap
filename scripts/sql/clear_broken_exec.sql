-- Delete packages with missing or invalid binaries
DELETE FROM packages
WHERE install_path NOT LIKE '%/bin/actual/%'  -- Check path format
   OR install_path NOT LIKE '%-%-%'  -- Check expected naming pattern
   OR binary_name = ''  -- Empty string
   OR trim(binary_name) = '';  -- Whitespace only