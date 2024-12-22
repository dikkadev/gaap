#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

check_db

echo -e "${YELLOW}Checking for broken packages...${NC}"
# Store the output in a variable
output=$(sqlite3 "$DB_PATH" < "$SCRIPTS_DIR/sql/clear_broken.sql")

if [ -z "$output" ]; then
    echo -e "${GREEN}No broken packages found.${NC}"
    exit 0
fi

echo -e "${YELLOW}The following packages will be removed:${NC}"
print_table "clear_broken.sql"

confirm_action "Do you want to remove these packages?" && {
    execute_sql "clear_broken_exec.sql"
    echo -e "${GREEN}Broken packages have been removed.${NC}"
} 