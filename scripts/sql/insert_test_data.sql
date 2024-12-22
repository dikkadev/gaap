-- Clear existing data
DELETE FROM packages;

-- Insert some test packages
INSERT INTO packages (owner, repo, version, install_path, binary_name, frozen, installed_at, updated_at)
VALUES 
    ('cli', 'cli', 'v2.40.0', '/home/user/gaap/bin/actual/cli-cli-v2.40.0', 'gh', 0, 
     datetime('now', '-60 days'), datetime('now', '-60 days')),
    
    ('charmbracelet', 'gum', 'v0.11.0', '/home/user/gaap/bin/actual/charmbracelet-gum-v0.11.0', 'gum', 1,
     datetime('now', '-30 days'), datetime('now', '-30 days')),
    
    ('charmbracelet', 'glow', 'v1.5.1', '/home/user/gaap/bin/actual/charmbracelet-glow-v1.5.1', 'glow', 0,
     datetime('now', '-45 days'), datetime('now', '-15 days')),
    
    ('junegunn', 'fzf', 'v0.44.1', '/home/user/gaap/bin/actual/junegunn-fzf-v0.44.1', 'fzf', 0,
     datetime('now', '-90 days'), datetime('now', '-90 days')),
    
    ('BurntSushi', 'ripgrep', 'v14.0.3', '/home/user/gaap/bin/actual/BurntSushi-ripgrep-v14.0.3', 'rg', 1,
     datetime('now', '-10 days'), datetime('now', '-10 days')),
    
    ('sharkdp', 'bat', 'v0.24.0', '/home/user/gaap/bin/actual/sharkdp-bat-v0.24.0', 'bat', 0,
     datetime('now', '-120 days'), datetime('now', '-120 days')); 