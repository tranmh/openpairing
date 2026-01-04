#!/bin/bash
# Script to merge branches following best practices with merge commits
# Usage: merge_branch.sh <source-branch> <target-branch>
#
# Example: merge_branch.sh feature/USER-123-add-login develop

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Validate arguments
if [ "$#" -ne 2 ]; then
    echo -e "${RED}Usage: merge_branch.sh <source-branch> <target-branch>${NC}"
    echo ""
    echo "Example: merge_branch.sh feature/USER-123-add-login develop"
    exit 1
fi

SOURCE_BRANCH=$1
TARGET_BRANCH=$2

echo -e "${YELLOW}=== Git Merge Workflow ===${NC}"
echo ""
echo "Source branch: $SOURCE_BRANCH"
echo "Target branch: $TARGET_BRANCH"
echo ""

# Check if source branch exists
if ! git rev-parse --verify "$SOURCE_BRANCH" > /dev/null 2>&1; then
    echo -e "${RED}Error: Source branch '$SOURCE_BRANCH' does not exist${NC}"
    exit 1
fi

# Check if target branch exists
if ! git rev-parse --verify "$TARGET_BRANCH" > /dev/null 2>&1; then
    echo -e "${RED}Error: Target branch '$TARGET_BRANCH' does not exist${NC}"
    exit 1
fi

# Checkout target branch
echo -e "${YELLOW}Step 1: Checking out target branch${NC}"
git checkout "$TARGET_BRANCH"

# Pull latest changes
echo -e "${YELLOW}Step 2: Pulling latest changes from origin/$TARGET_BRANCH${NC}"
git pull origin "$TARGET_BRANCH"

# Merge with --no-ff to create merge commit
echo -e "${YELLOW}Step 3: Merging $SOURCE_BRANCH into $TARGET_BRANCH${NC}"
echo ""

if git merge --no-ff "$SOURCE_BRANCH" -m "Merge branch '$SOURCE_BRANCH' into $TARGET_BRANCH"; then
    echo ""
    echo -e "${GREEN}✅ Merge successful!${NC}"
    echo ""

    # Ask if user wants to push
    echo -e "${YELLOW}Push changes to origin/$TARGET_BRANCH? (y/n)${NC}"
    read -r response

    if [[ "$response" =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Step 4: Pushing to origin/$TARGET_BRANCH${NC}"
        git push origin "$TARGET_BRANCH"
        echo -e "${GREEN}✅ Successfully pushed to origin/$TARGET_BRANCH${NC}"
        echo ""

        # Ask if user wants to delete source branch
        echo -e "${YELLOW}Delete source branch '$SOURCE_BRANCH'? (y/n)${NC}"
        read -r delete_response

        if [[ "$delete_response" =~ ^[Yy]$ ]]; then
            echo -e "${YELLOW}Step 5: Deleting local branch $SOURCE_BRANCH${NC}"
            git branch -d "$SOURCE_BRANCH"

            echo -e "${YELLOW}Step 6: Deleting remote branch origin/$SOURCE_BRANCH${NC}"
            git push origin --delete "$SOURCE_BRANCH"

            echo ""
            echo -e "${GREEN}✅ Branch cleanup complete!${NC}"
        else
            echo ""
            echo -e "${YELLOW}Branch '$SOURCE_BRANCH' kept. Delete manually with:${NC}"
            echo "  git branch -d $SOURCE_BRANCH"
            echo "  git push origin --delete $SOURCE_BRANCH"
        fi
    else
        echo ""
        echo -e "${YELLOW}Changes not pushed. Push manually with:${NC}"
        echo "  git push origin $TARGET_BRANCH"
    fi

    echo ""
    echo -e "${GREEN}=== Merge Complete ===${NC}"
else
    echo ""
    echo -e "${RED}❌ Merge failed - conflicts detected${NC}"
    echo ""
    echo -e "${YELLOW}To resolve conflicts:${NC}"
    echo "  1. Open conflicted files and resolve conflicts"
    echo "  2. Stage resolved files: git add <file>"
    echo "  3. Complete merge: git commit"
    echo "  4. Push changes: git push origin $TARGET_BRANCH"
    echo ""
    echo -e "${YELLOW}To abort merge:${NC}"
    echo "  git merge --abort"
    exit 1
fi
