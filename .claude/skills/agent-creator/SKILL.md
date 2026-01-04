---
name: agent-creator
description: Create Claude Code subagents following official Anthropic best practices. Use when creating agents, subagents, custom agents, Task tool agents, or specialized AI assistants. Generates agent files in .claude/agents/ with proper YAML frontmatter and system prompts.
---

# Agent Creator Skill

## Purpose

Create specialized Claude Code subagents following official Anthropic documentation and best practices. Agents are stored as markdown files with YAML frontmatter in `.claude/agents/` and can be invoked via the Task tool.

## When to Use

Automatically activates when you mention:
- Creating agents or subagents
- Custom agents for Claude Code
- Task tool agents
- Specialized AI assistants
- Agent configuration
- `.claude/agents/` directory

## Agent File Structure

Agents are markdown files stored in `.claude/agents/{agent-name}.md`:

```markdown
---
name: agent-identifier
description: Natural language description for auto-activation
tools: Read, Glob, Grep, Bash
model: sonnet
permissionMode: default
---

[System prompt content here]
```

## Required Fields

| Field | Description |
|-------|-------------|
| `name` | Lowercase with hyphens, must match filename |
| `description` | When/why to use; triggers auto-activation |

## Optional Fields

| Field | Default | Options |
|-------|---------|---------|
| `tools` | all | Comma-separated: `Read, Edit, Write, Glob, Grep, Bash, WebFetch, WebSearch` |
| `model` | inherit | `haiku` (fast/cheap), `sonnet` (balanced), `opus` (powerful), `inherit` |
| `permissionMode` | default | `default`, `acceptEdits`, `bypassPermissions`, `plan`, `ignore` |
| `skills` | none | Comma-separated skill names to auto-load |

## Best Practices

### 1. Single Purpose
Design each agent with ONE focused responsibility. Avoid multi-purpose agents.

### 2. Minimal Tool Access
Grant ONLY necessary tools:
- Improves security
- Better focus and performance
- Reduces context noise

### 3. Detailed System Prompts
Include:
- Clear role definition
- Specific responsibilities
- Step-by-step approach
- Constraints and limitations
- Output format specifications

### 4. Description for Auto-Activation
Use phrases like "PROACTIVELY" or "MUST BE USED" for important agents:
```yaml
description: Senior code reviewer. PROACTIVELY use after significant code changes.
```

### 5. Model Selection
- `haiku`: Quick analysis, simple tasks (3x cost savings)
- `sonnet`: General-purpose, balanced (default)
- `opus`: Complex reasoning, detailed analysis

## Creating an Agent

### Step 1: Define Purpose
What specific task does this agent handle?

### Step 2: Choose Tools
What tools does the agent need? Less is more.

### Step 3: Write System Prompt
Follow the template structure in [AGENT_TEMPLATE.md](AGENT_TEMPLATE.md)

### Step 4: Save to .claude/agents/
File: `.claude/agents/{agent-name}.md`

### Step 5: Test
Invoke with: "Use the {agent-name} agent to help with {task}"

## Quick Examples

### Research Agent
```markdown
---
name: code-explorer
description: Explore and understand codebases. Find patterns, architecture, and answer questions about code structure.
tools: Read, Glob, Grep
model: haiku
---

You are a codebase exploration specialist...
```

### Code Reviewer Agent
```markdown
---
name: senior-reviewer
description: PROACTIVELY review code after changes. Focus on bugs, security, and best practices.
tools: Read, Glob, Grep, Bash
model: sonnet
---

You are a senior code reviewer...
```

### Debugging Agent
```markdown
---
name: bug-hunter
description: Debug issues and find root causes. Analyze errors, trace execution, suggest fixes.
tools: Read, Glob, Grep, Bash, Edit
model: sonnet
---

You are a debugging specialist...
```

## Output Location

All agents are created in:
```
.claude/agents/{agent-name}.md
```

This location is:
- Project-level (shared via git)
- Takes precedence over user-level agents
- Automatically discovered by Claude Code

## Reference

See [AGENT_TEMPLATE.md](AGENT_TEMPLATE.md) for the complete agent template with all sections and examples.

---

**Storage**: `.claude/agents/`
**Format**: Markdown with YAML frontmatter
**Invocation**: Automatic (via description) or explicit ("Use the X agent")
