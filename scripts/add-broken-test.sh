#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

check_db

echo -e "${YELLOW}Adding broken test packages...${NC}"
execute_sql "insert_broken_test_data.sql"
echo -e "${GREEN}Broken test data has been added.${NC}"

echo -e "\n${YELLOW}All packages (including broken):${NC}"
print_table "list_packages.sql"

echo -e "\n${YELLOW}Checking for broken packages:${NC}"
print_table "clear_broken.sql" 