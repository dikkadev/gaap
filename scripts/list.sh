#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

check_db

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --frozen)
            SHOW_FROZEN=1
            shift
            ;;
        --outdated)
            SHOW_OUTDATED=1
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

if [ "$SHOW_FROZEN" = "1" ]; then
    print_table "list_frozen.sql"
elif [ "$SHOW_OUTDATED" = "1" ]; then
    print_table "list_outdated.sql"
else
    print_table "list_packages.sql"
fi 