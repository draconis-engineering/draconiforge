#!/usr/bin/env bash

# Loop through all items in the current directory that end with a trailing slash (directories only)
for dir in ./*/; do
    # Strip the trailing slash for cleaner printing
    dir_name="${dir%/}"

    # Check if it contains a .git folder to verify it is a Git repository
    if [ -d "$dir/.git" ]; then
        echo "========================================"
        echo "Updating: $dir_name"
        echo "========================================"

        # Run git pull inside the target directory without permanently changing your shell's location
        git -C "$dir" pull

        echo "" # Add a blank line for readability
    else
        echo "Skipping: $dir_name (Not a Git repository)"
    fi
done
