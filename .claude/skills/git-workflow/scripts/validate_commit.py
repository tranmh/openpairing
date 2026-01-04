#!/usr/bin/env python3
"""
Validate commit messages against Conventional Commits specification.
Can be used as a git commit-msg hook or standalone validator.

Usage:
    python validate_commit.py <commit-message-file>
    python validate_commit.py --message "feat(auth): add login"
"""

import re
import sys
import argparse


# Conventional commit types
VALID_TYPES = [
    'feat', 'fix', 'docs', 'style', 'refactor',
    'perf', 'test', 'build', 'ci', 'chore'
]

# Regex pattern for conventional commit
COMMIT_PATTERN = re.compile(
    r'^(?P<type>' + '|'.join(VALID_TYPES) + r')'
    r'(?:\((?P<scope>[a-z0-9-]+)\))?'
    r'(?P<breaking>!)?'
    r': '
    r'(?P<subject>.+)$',
    re.IGNORECASE
)


def validate_commit_message(message):
    """
    Validate a commit message against Conventional Commits format.

    Returns:
        tuple: (is_valid, error_message)
    """
    lines = message.strip().split('\n')

    if not lines:
        return False, "Commit message is empty"

    # Get the first line (subject)
    subject = lines[0].strip()

    # Skip merge commits
    if subject.startswith('Merge '):
        return True, None

    # Check if it matches the pattern
    match = COMMIT_PATTERN.match(subject)

    if not match:
        return False, (
            f"Commit message does not follow Conventional Commits format.\n"
            f"\n"
            f"Expected format: <type>(<scope>): <subject>\n"
            f"\n"
            f"Examples:\n"
            f"  feat(auth): add login functionality\n"
            f"  fix(api): resolve null pointer exception\n"
            f"  docs(readme): update installation guide\n"
            f"\n"
            f"Valid types: {', '.join(VALID_TYPES)}\n"
            f"\n"
            f"Your message: {subject}"
        )

    # Validate subject
    commit_type = match.group('type').lower()
    scope = match.group('scope')
    subject_text = match.group('subject')

    # Check type is valid
    if commit_type not in VALID_TYPES:
        return False, (
            f"Invalid commit type '{commit_type}'.\n"
            f"Valid types: {', '.join(VALID_TYPES)}"
        )

    # Check subject is not empty
    if not subject_text:
        return False, "Subject cannot be empty"

    # Check subject doesn't end with period
    if subject_text.endswith('.'):
        return False, "Subject should not end with a period"

    # Check subject length (recommended < 50 chars, max 72)
    if len(subject) > 72:
        return False, f"Subject line is too long ({len(subject)} chars). Keep it under 72 characters."

    # Check subject starts with lowercase
    if subject_text[0].isupper():
        return False, "Subject should start with lowercase letter"

    # Check for imperative mood (basic check)
    forbidden_endings = ['ed', 'ing']
    first_word = subject_text.split()[0].lower()
    if any(first_word.endswith(end) for end in forbidden_endings):
        return False, (
            f"Subject should use imperative mood (e.g., 'add' not 'added' or 'adding').\n"
            f"Your first word: '{first_word}'"
        )

    # All checks passed
    return True, None


def main():
    parser = argparse.ArgumentParser(
        description='Validate commit messages against Conventional Commits'
    )
    parser.add_argument(
        'file',
        nargs='?',
        help='Path to commit message file (used in git hooks)'
    )
    parser.add_argument(
        '--message', '-m',
        help='Commit message string to validate'
    )

    args = parser.parse_args()

    # Get commit message
    if args.message:
        commit_message = args.message
    elif args.file:
        try:
            with open(args.file, 'r') as f:
                commit_message = f.read()
        except FileNotFoundError:
            print(f"Error: File '{args.file}' not found", file=sys.stderr)
            sys.exit(1)
    else:
        parser.print_help()
        sys.exit(1)

    # Validate
    is_valid, error = validate_commit_message(commit_message)

    if is_valid:
        print("✅ Commit message is valid")
        sys.exit(0)
    else:
        print("❌ Invalid commit message:", file=sys.stderr)
        print(f"\n{error}\n", file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
