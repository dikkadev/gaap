#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

check_db

echo -e "${YELLOW}This will clear all existing data and add test data.${NC}"
confirm_action "Do you want to continue?" && {
    execute_sql "insert_test_data.sql"
    echo -e "${GREEN}Test data has been added successfully.${NC}"
    echo -e "\n${YELLOW}Current packages:${NC}"
    print_table "list_packages.sql"
} 