-- Insert broken test packages
INSERT INTO packages (owner, repo, version, install_path, binary_name, frozen, installed_at, updated_at)
VALUES 
    -- Wrong path format (not in bin/actual)
    ('broken', 'wrong-path', 'v1.0.0', '/usr/local/bin/something', 'something', 0, 
     datetime('now'), datetime('now')),
    
    -- Empty binary name (but not NULL)
    ('broken', 'no-binary', 'v1.0.0', '/home/user/gaap/bin/actual/broken-no-binary-v1.0.0', '', 0,
     datetime('now'), datetime('now')),
    
    -- Wrong naming pattern in path
    ('broken', 'bad-pattern', 'v1.0.0', '/home/user/gaap/bin/actual/just-a-file', 'bad', 0,
     datetime('now'), datetime('now')),
    
    -- Invalid binary name (just spaces)
    ('broken', 'spaces-binary', 'v1.0.0', '/home/user/gaap/bin/actual/broken-spaces-binary-v1.0.0', '   ', 0,
     datetime('now'), datetime('now')),
    
    -- Multiple issues (wrong path and pattern)
    ('broken', 'multiple-issues', 'v1.0.0', '/tmp/random-file', 'multiple', 0,
     datetime('now'), datetime('now')); 