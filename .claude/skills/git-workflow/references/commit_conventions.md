# Conventional Commits Reference

## Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

## Components

### Type (Required)

The type indicates the nature of the change:

- **feat**: A new feature for the user
  - Example: `feat(auth): add OAuth2 login support`

- **fix**: A bug fix
  - Example: `fix(api): handle null pointer in user service`

- **docs**: Documentation only changes
  - Example: `docs(readme): update installation instructions`

- **style**: Changes that don't affect code meaning (formatting, whitespace)
  - Example: `style(css): fix indentation in main.css`

- **refactor**: Code change that neither fixes a bug nor adds a feature
  - Example: `refactor(auth): extract validation logic to separate service`

- **perf**: Performance improvements
  - Example: `perf(database): add index on user_id column`

- **test**: Adding or updating tests
  - Example: `test(auth): add unit tests for login service`

- **build**: Changes to build system or dependencies
  - Example: `build(deps): upgrade react to v18.2.0`

- **ci**: Changes to CI/CD configuration
  - Example: `ci(github): add automated deployment workflow`

- **chore**: Other changes that don't modify src or test files
  - Example: `chore(gitignore): add .env to gitignore`

### Scope (Optional but Recommended)

The scope provides context about what part of the codebase is affected:

- Use component/module names: `auth`, `api`, `database`, `ui`
- Use feature names: `login`, `payment`, `notifications`
- Use file/folder names: `middleware`, `utils`, `config`

Examples:
- `feat(auth): add 2FA support`
- `fix(payment): resolve timeout on checkout`
- `refactor(database): migrate to connection pooling`

### Subject (Required)

The subject is a brief description of the change:

- Use imperative mood: "add" not "added" or "adds"
- Don't capitalize first letter
- No period at the end
- Keep it under 50 characters
- Be clear and specific

**Good examples:**
- `add user authentication`
- `fix null pointer exception in payment service`
- `update dependencies to latest versions`

**Bad examples:**
- `Added user authentication` (wrong tense)
- `Fix bug` (too vague)
- `Updated the dependencies to the latest versions available.` (too long, capitalized, period)

### Body (Optional)

The body provides additional context:

- Use imperative mood
- Wrap at 72 characters
- Explain what and why, not how
- Separate from subject with blank line

Example:
```
feat(auth): add OAuth2 login support

Users can now authenticate using Google and GitHub accounts.
This reduces friction in the signup process and improves
security by delegating authentication to trusted providers.
```

### Footer (Optional)

The footer contains metadata:

- **Breaking changes**: Start with `BREAKING CHANGE:`
- **Issue references**: `Closes #123`, `Fixes #456`, `Refs #789`

Example:
```
feat(api): change user endpoint response format

BREAKING CHANGE: The /api/users endpoint now returns an array
of user objects instead of a paginated response object.

Closes #234
```

## Examples

### Simple Feature
```
feat(dashboard): add revenue chart widget
```

### Bug Fix with Details
```
fix(auth): prevent token refresh loop

The token refresh mechanism was triggering multiple times
when the token expired, causing performance issues. Added
a mutex to ensure only one refresh happens at a time.

Fixes #567
```

### Breaking Change
```
feat(api): migrate to REST v2 endpoints

BREAKING CHANGE: All v1 API endpoints are deprecated and
will be removed in the next major release. Clients must
update to use v2 endpoints with the new authentication flow.

Refs #890
```

### Refactoring
```
refactor(database): extract query builder to separate class

Improves testability and reusability of database queries.
```

### Documentation
```
docs(contributing): add code review guidelines
```

### Performance Improvement
```
perf(search): implement search results caching

Reduces average search response time from 500ms to 50ms
by caching frequently searched queries for 5 minutes.
```

## Commit Message Best Practices

1. **Make atomic commits**: Each commit should represent one logical change
2. **Commit often**: Small, frequent commits are better than large, infrequent ones
3. **Write in imperative mood**: "add feature" not "added feature"
4. **Be specific**: "fix login validation" not "fix bug"
5. **Reference issues**: Always include ticket/issue numbers when applicable
6. **Keep subject short**: Under 50 characters when possible
7. **Use body for details**: Explain the why and what, not the how
8. **Test before committing**: Ensure the code works

## Quick Reference

| Type | When to Use | Example |
|------|-------------|---------|
| feat | New feature | `feat(auth): add password reset` |
| fix | Bug fix | `fix(api): handle timeout errors` |
| docs | Documentation | `docs(readme): update setup guide` |
| style | Formatting | `style: fix code indentation` |
| refactor | Code restructuring | `refactor(utils): simplify helper functions` |
| perf | Performance | `perf(db): optimize query performance` |
| test | Tests | `test(auth): add login integration tests` |
| build | Build/dependencies | `build: update webpack config` |
| ci | CI/CD | `ci: add test coverage reporting` |
| chore | Maintenance | `chore: update .gitignore` |
