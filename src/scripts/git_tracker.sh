#!/usr/bin/env bash

# Set the target folder (defaults to current directory if empty)
TARGET_DIR="${1:-.}"

# Find all directories containing a .git folder (up to 2 levels deep)
find "$TARGET_DIR" -maxdepth 2 -type d -name ".git" | while read -r git_dir; do

    # Get the parent folder path of the .git folder
    repo_dir=$(dirname "$git_dir")
    repo_name=$(basename "$repo_dir")

    # Move into the repository context
    cd "$repo_dir" || continue

    # Get the current branch name safely
    branch=$(git branch --show-current 2>/dev/null)

    # Check for uncommitted changes or untracked files
    status=$(git status --short 2>/dev/null)

    # If there are uncommitted changes or untracked files, display a warning
    if [ -n "$status" ]; then

        # Display the repository name and branch, then the status output
        echo -e " [\033[1;31mX\033[0m] [\033[1;33m$repo_name\033[0m] ($branch) has uncommitted changes:"

        # Indent the status output for better readability
        echo "$status" | sed 's/^/   /'
    else
        echo -e " [\033[1;32m*\033[0m] [\033[1;32m$repo_name\033[0m] ($branch) Clean"
    fi

    # Return to starting directory
    cd - > /dev/null || exit
done
