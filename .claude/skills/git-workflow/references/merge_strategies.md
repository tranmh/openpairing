# Merge Strategies Reference

## Merge Commit Strategy (Recommended)

The merge commit strategy creates a merge commit that preserves the complete history of both branches. This is the recommended approach for this workflow.

### Characteristics

- Preserves full commit history from feature branch
- Creates a merge commit to tie branches together
- Non-linear history showing branch structure
- Easy to see where features were developed
- Easy to revert entire features by reverting the merge commit

### When to Use

- When you want to preserve detailed development history
- When you need to revert entire features easily
- When branch context is important for understanding changes
- For all merges in this workflow (feature → develop, hotfix → main/develop, etc.)

### Command

```bash
git merge --no-ff <branch-name>
```

The `--no-ff` flag forces creation of a merge commit even if fast-forward is possible.

### Example

```bash
# Merging feature branch to develop
git checkout develop
git pull origin develop
git merge --no-ff feature/USER-123-add-login
git push origin develop
```

### Resulting History

```
*   Merge branch 'feature/USER-123-add-login' into develop
|\
| * feat(auth): add login form validation
| * feat(auth): implement login API integration
| * feat(auth): add login UI components
|/
* Previous commit on develop
```

## Alternative Strategies (Not Used in This Workflow)

### Squash and Merge

Combines all commits from the feature branch into a single commit.

**Pros:**
- Clean, linear history
- One commit per feature
- Easy to read git log

**Cons:**
- Loses detailed development history
- Can't see individual commit messages from feature branch
- Harder to debug when you need to find when a specific change was introduced

**Command:**
```bash
git merge --squash <branch-name>
git commit -m "feat(scope): feature description"
```

### Rebase and Merge

Replays commits from feature branch on top of target branch without creating merge commit.

**Pros:**
- Linear history without merge commits
- Preserves individual commits
- Clean timeline

**Cons:**
- Rewrites history (dangerous for shared branches)
- Loses branch context
- More complex to understand for beginners
- Can create confusion if multiple people work on same branch

**Command:**
```bash
git checkout feature/USER-123-add-login
git rebase develop
git checkout develop
git merge feature/USER-123-add-login
```

## Best Practices for Merge Commits

### 1. Always Use --no-ff for Feature Merges

```bash
# Good
git merge --no-ff feature/USER-123-add-login

# Bad (allows fast-forward)
git merge feature/USER-123-add-login
```

### 2. Write Descriptive Merge Commit Messages

Default merge message is usually good, but can be improved:

```bash
# Default (acceptable)
Merge branch 'feature/USER-123-add-login' into develop

# Better (provides context)
Merge branch 'feature/USER-123-add-login' into develop

Adds OAuth2 authentication with Google and GitHub providers.
Includes login UI, API integration, and session management.

Closes #123
```

### 3. Resolve Conflicts Carefully

When merge conflicts occur:

```bash
# Start merge
git merge --no-ff feature/USER-123-add-login

# Conflicts occur - resolve them in your editor
# Look for conflict markers: <<<<<<<, =======, >>>>>>>

# After resolving conflicts
git add <resolved-files>
git commit  # Complete the merge
```

### 4. Pull Before Merging

Always ensure target branch is up-to-date:

```bash
# Good workflow
git checkout develop
git pull origin develop  # Get latest changes
git merge --no-ff feature/USER-123-add-login

# Bad workflow (might create issues)
git checkout develop
git merge --no-ff feature/USER-123-add-login  # Might be outdated
```

### 5. Clean Up After Merge

Delete the feature branch after successful merge:

```bash
# Delete local branch
git branch -d feature/USER-123-add-login

# Delete remote branch
git push origin --delete feature/USER-123-add-login
```

## Handling Special Cases

### Merging Hotfix to Multiple Branches

Hotfixes need to merge to both main and develop:

```bash
# Merge to main
git checkout main
git pull origin main
git merge --no-ff hotfix/CRIT-999-security-patch
git push origin main

# Tag the release
git tag -a v1.2.1 -m "Hotfix: Security patch"
git push origin v1.2.1

# Merge to develop
git checkout develop
git pull origin develop
git merge --no-ff hotfix/CRIT-999-security-patch
git push origin develop

# Clean up
git branch -d hotfix/CRIT-999-security-patch
git push origin --delete hotfix/CRIT-999-security-patch
```

### Merging Release Branch

Release branches merge to both main and develop:

```bash
# Merge to main
git checkout main
git merge --no-ff release/v1.3.0
git tag -a v1.3.0 -m "Release v1.3.0"
git push origin main --tags

# Merge to develop
git checkout develop
git merge --no-ff release/v1.3.0
git push origin develop

# Clean up
git branch -d release/v1.3.0
git push origin --delete release/v1.3.0
```

## Pull Request Workflow

When using pull requests (recommended):

1. **Create PR**: Create pull request from feature branch to target branch
2. **Review**: Team reviews code changes
3. **Approve**: PR is approved by required reviewers
4. **Merge**: Use "Create a merge commit" option (not squash or rebase)
5. **Delete**: Automatically delete feature branch after merge

### GitHub PR Merge Settings

Configure repository to use merge commits:

1. Go to Settings → General → Pull Requests
2. Uncheck "Allow squash merging"
3. Uncheck "Allow rebase merging"
4. Check "Allow merge commits"
5. Check "Automatically delete head branches"

## Visualization

### Merge Commit Graph

```
* (develop) Merge feature/USER-456-add-dashboard
|\
| * (feature/USER-456-add-dashboard) feat(dashboard): add charts
| * feat(dashboard): add data fetching
|/
* (develop) Merge feature/USER-123-add-login
|\
| * (feature/USER-123-add-login) feat(auth): add login validation
| * feat(auth): add login UI
|/
* (develop) Previous commit
```

This visualization shows:
- Clear branch points and merge points
- Feature development isolation
- Complete commit history preserved
- Easy to identify which commits belong to which feature
