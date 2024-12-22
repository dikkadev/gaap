#!/bin/bash

# Common variables
DB_PATH="$HOME/gaap/db/gaap.db"
SCRIPTS_DIR="$(dirname "${BASH_SOURCE[0]}")"

# Colors and styles
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if database exists
check_db() {
    if [ ! -f "$DB_PATH" ]; then
        echo -e "${RED}Error: Database not found at $DB_PATH${NC}"
        exit 1
    fi
}

# Confirm action using gum
confirm_action() {
    local message="$1"
    gum confirm "$message" || exit 0
}

# Execute SQL file
execute_sql() {
    local sql_file="$1"
    local full_path="$SCRIPTS_DIR/sql/$sql_file"
    
    if [ ! -f "$full_path" ]; then
        echo -e "${RED}Error: SQL file not found: $full_path${NC}"
        exit 1
    fi
    
    sqlite3 "$DB_PATH" < "$full_path"
}

# Print table output with headers
print_table() {
    local sql_file="$1"
    local full_path="$SCRIPTS_DIR/sql/$sql_file"
    
    if [ ! -f "$full_path" ]; then
        echo -e "${RED}Error: SQL file not found: $full_path${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}Results:${NC}"
    sqlite3 -header -column "$DB_PATH" < "$full_path"
} 