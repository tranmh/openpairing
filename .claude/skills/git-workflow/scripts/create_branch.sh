#!/bin/bash
# Script to create a new branch following naming conventions
# Usage: create_branch.sh <type> <ticket-id> <description>
#
# Example: create_branch.sh feature USER-123 add-login

set -e

# Validate arguments
if [ "$#" -ne 3 ]; then
    echo "Usage: create_branch.sh <type> <ticket-id> <description>"
    echo ""
    echo "Types: feature, bugfix, hotfix, refactor"
    echo "Example: create_branch.sh feature USER-123 add-login"
    exit 1
fi

TYPE=$1
TICKET=$2
DESCRIPTION=$3

# Validate branch type
case $TYPE in
    feature|bugfix|refactor)
        BASE_BRANCH="develop"
        ;;
    hotfix)
        BASE_BRANCH="main"
        ;;
    *)
        echo "Error: Invalid branch type '$TYPE'"
        echo "Valid types: feature, bugfix, hotfix, refactor"
        exit 1
        ;;
esac

# Construct branch name
BRANCH_NAME="${TYPE}/${TICKET}-${DESCRIPTION}"

# Convert to lowercase and replace spaces/underscores with hyphens
BRANCH_NAME=$(echo "$BRANCH_NAME" | tr '[:upper:]' '[:lower:]' | tr '_ ' '--')

echo "Creating branch: $BRANCH_NAME"
echo "Base branch: $BASE_BRANCH"
echo ""

# Ensure we're on the base branch and it's up to date
echo "Checking out $BASE_BRANCH..."
git checkout "$BASE_BRANCH"

echo "Pulling latest changes from origin/$BASE_BRANCH..."
git pull origin "$BASE_BRANCH"

# Create and checkout new branch
echo "Creating and checking out $BRANCH_NAME..."
git checkout -b "$BRANCH_NAME"

echo ""
echo "✅ Successfully created branch: $BRANCH_NAME"
echo ""
echo "Next steps:"
echo "  1. Make your changes"
echo "  2. Commit with conventional commit messages"
echo "  3. Push with: git push -u origin $BRANCH_NAME"
echo "  4. Create a pull request to merge back to $BASE_BRANCH"
