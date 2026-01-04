#!/bin/bash
set -e

# Debug: Log that hook was called
echo "[DEBUG] pre-tool-use.sh called at $(date)" >>/tmp/pre-tool-use-debug.log

# Read tool information from stdin
tool_info=$(cat)

# Debug: Log the input
echo "[DEBUG] tool_info: $tool_info" >>/tmp/pre-tool-use-debug.log

# Extract relevant data
tool_name=$(echo "$tool_info" | jq -r '.tool_name // empty')
file_path=$(echo "$tool_info" | jq -r '.tool_input.file_path // empty')
bash_command=$(echo "$tool_info" | jq -r '.tool_input.command // empty')
transcript_path=$(echo "$tool_info" | jq -r '.transcript_path // empty')

# ============================================================================
# Git Workflow Detection - Check for git operations requiring git-workflow skill
# ============================================================================
if [[ "$tool_name" == "Bash" ]] && [[ -n "$bash_command" ]]; then
  echo "[DEBUG] Bash command detected: $bash_command" >>/tmp/pre-tool-use-debug.log

  # Check if command contains git commit, git merge, or git branch operations
  if [[ "$bash_command" =~ git[[:space:]]+(commit|merge|branch) ]]; then
    echo "[DEBUG] Git operation detected - checking if git-workflow skill is active" >>/tmp/pre-tool-use-debug.log

    # Check if git-workflow skill is currently active by looking at the transcript
    if [[ -n "$transcript_path" ]] && [[ -f "$transcript_path" ]]; then
      # Check if git-workflow skill was recently invoked in this session (check last 100 lines)
      if tail -100 "$transcript_path" 2>/dev/null | grep -q '"skill".*"git-workflow"\|"command-name":"git-workflow"'; then
        echo "[DEBUG] git-workflow skill is active - allowing git operation" >>/tmp/pre-tool-use-debug.log
        exit 0 # Allow the tool to proceed
      fi
    fi

    echo "[DEBUG] git-workflow skill NOT active - blocking and injecting skill" >>/tmp/pre-tool-use-debug.log

    # Return JSON to deny the tool use and inform Claude to use git-workflow skill
    cat <<'EOF'
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "Git operation detected (commit/merge/branch). You must use the 'git-workflow' skill before performing git operations. This skill ensures proper git workflow with conventional commits, proper branching strategies, and clean repository history. Please invoke the skill using: Skill tool with skill parameter 'git-workflow'"
  }
}
EOF

    echo "[DEBUG] Blocked and injected git-workflow skill requirement" >>/tmp/pre-tool-use-debug.log
    exit 0 # Exit 0 when returning JSON
  fi
fi

# ============================================================================
# Markdown File Detection - Check for markdown creation requiring docs-organizer
# ============================================================================
if [[ "$tool_name" == "Write" ]] && [[ "$file_path" =~ \.(md|markdown)$ ]]; then
  echo "[DEBUG] Markdown file detected - checking if it's in .claude/ directory" >>/tmp/pre-tool-use-debug.log
  # Exception: Allow all markdown files inside .claude/ directory
  if [[ "$file_path" =~ \.claude/ ]]; then
    echo "[DEBUG] Markdown file in .claude/ directory - allowing without docs-organizer" >>/tmp/pre-tool-use-debug.log
    exit 0 # Allow .claude/ markdown files to be created
  fi

  # Check if docs-organizer skill is currently active by looking at the transcript
  if [[ -n "$transcript_path" ]] && [[ -f "$transcript_path" ]]; then
    # Check if docs-organizer skill was recently invoked in this session (check last 100 lines)
    if tail -100 "$transcript_path" 2>/dev/null | grep -q '"skill".*"docs-organizer"\|"command-name":"docs-organizer"'; then
      echo "[DEBUG] docs-organizer skill is active - allowing markdown creation" >>/tmp/pre-tool-use-debug.log
      exit 0 # Allow the tool to proceed
    fi
  fi

  echo "[DEBUG] docs-organizer skill NOT active - blocking and injecting skill" >>/tmp/pre-tool-use-debug.log

  # Return JSON to deny the tool use and inform Claude to use docs-organizer skill
  cat <<'EOF'
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "Markdown file creation detected. You must use the 'docs-organizer' skill before creating markdown files. This skill will help organize the documentation properly in the correct subdirectory under docs/. Please invoke the skill using: Skill tool with skill parameter 'docs-organizer'"
  }
}
EOF

  echo "[DEBUG] Blocked and injected docs-organizer skill requirement" >>/tmp/pre-tool-use-debug.log
  exit 0 # Exit 0 when returning JSON
fi

# Exit 0 to allow the tool to proceed
exit 0
