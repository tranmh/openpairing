---
name: docs-organizer
description: This skill should be used whenever Claude is about to create or write a new markdown (.md) file. It automatically organizes markdown files into appropriate subdirectories under the docs/ directory based on the file's purpose and content type, ensuring consistent documentation structure across software engineering projects.
---

# Docs Organizer

## Overview

This skill ensures that all markdown documentation files are systematically organized into appropriate subdirectories under the `docs/` directory. It analyzes the filename and intended content to determine the correct category, creates the necessary directory structure, and writes files to their proper location.

## When to Use This Skill

**ALWAYS** use this skill before creating or writing any new markdown file (.md extension). This includes:
- Development plans, roadmaps, and project documentation
- Code reviews, design reviews, and technical assessments
- Architecture documents, system designs, and ADRs
- API documentation and specifications
- Meeting notes, retrospectives, and team communications
- Any other markdown documentation files

**Exception:** Do not use this skill for:
- README.md files in the project root or package directories
- CHANGELOG.md files at the project root
- LICENSE.md or CONTRIBUTING.md files at the project root
- Markdown files that are part of application content (e.g., blog posts, user-facing content)

## Documentation Categories

Organize markdown files into the following subdirectories under `docs/`:

| Category | Directory | File Types |
|----------|-----------|------------|
| Development Plans | `docs/plan/` | DevelopmentPlan.md, ProjectPlan.md, Roadmap.md, Sprint*.md |
| Code Reviews | `docs/review/` | CodeReview*.md, PullRequestReview*.md, DesignReview*.md |
| Architecture | `docs/architecture/` | Architecture*.md, SystemDesign*.md, ADR*.md, TechStack*.md |
| API Documentation | `docs/api/` | API*.md, Endpoints*.md, OpenAPI*.md, RestAPI*.md |
| Requirements | `docs/requirements/` | Requirements*.md, UserStories*.md, FeatureSpec*.md, BRD*.md |
| Technical Design | `docs/design/` | Design*.md, TechnicalDesign*.md, UIDesign*.md, DataModel*.md |
| Meeting Notes | `docs/meetings/` | Meeting*.md, Standup*.md, Retrospective*.md, Notes*.md |
| Testing | `docs/testing/` | TestPlan*.md, TestResults*.md, QA*.md, TestStrategy*.md |
| Deployment | `docs/deployment/` | Deployment*.md, ReleaseNotes*.md, Runbook*.md, Operations*.md |
| Onboarding | `docs/onboarding/` | Onboarding*.md, TeamHandbook*.md, GettingStarted*.md, Setup*.md |
| Technical Specs | `docs/specs/` | Specification*.md, TechnicalSpec*.md, Protocol*.md, Standard*.md |
| Proposals | `docs/proposals/` | Proposal*.md, RFC*.md, ProjectProposal*.md |
| Changelog | `docs/changelog/` | CHANGELOG*.md, VersionHistory*.md, ReleaseHistory*.md |
| Troubleshooting | `docs/troubleshooting/` | Troubleshooting*.md, Debug*.md, FAQ*.md, KnownIssues*.md |
| Database | `docs/database/` | Schema*.md, Database*.md, Migration*.md, ERDiagram*.md |
| Guides | `docs/guides/` | Guide*.md, HowTo*.md, Tutorial*.md, Walkthrough*.md |
| Security | `docs/security/` | Security*.md, ThreatModel*.md, SecurityPolicy*.md, Audit*.md |
| Process | `docs/process/` | Process*.md, Workflow*.md, SOP*.md, Procedure*.md |
| General/Other | `docs/general/` | All uncategorized markdown files |

## Workflow

Follow these steps when creating any markdown file:

### 1. Analyze the Filename and Purpose

Examine the filename and intended content to determine which category best fits the document. Use pattern matching on the filename first, then consider the document's purpose if the filename is ambiguous.

**Pattern Matching Rules:**
- Match case-insensitively
- Check for category keywords in the filename
- If multiple categories match, use the most specific one
- If no pattern matches, ask yourself: "What is the primary purpose of this document?"

### 2. Determine the Target Directory

Based on the categorization, construct the full path:
```
docs/<category>/<filename>
```

**Examples:**
- `DevelopmentPlan.md` → `docs/plan/DevelopmentPlan.md`
- `CodeReview.md` → `docs/review/CodeReview.md`
- `APIEndpoints.md` → `docs/api/APIEndpoints.md`
- `SecurityAudit2024.md` → `docs/security/SecurityAudit2024.md`

### 3. Ensure Directory Structure Exists

Before writing the file, verify that the target directory exists. If it doesn't, create the full directory path including any intermediate directories.

```bash
mkdir -p docs/<category>
```

### 4. Write the File

Write the markdown file to the determined location under the docs/ directory.

## Category Selection Guidelines

When the filename is ambiguous or doesn't match a clear pattern:

**Ask these questions:**
1. Is this about planning or roadmapping? → `docs/plan/`
2. Is this a review of code, design, or architecture? → `docs/review/`
3. Is this about system architecture or design decisions? → `docs/architecture/`
4. Is this API or endpoint documentation? → `docs/api/`
5. Is this about requirements or specifications? → `docs/requirements/` or `docs/specs/`
6. Is this meeting notes or team communication? → `docs/meetings/`
7. Is this about testing or QA? → `docs/testing/`
8. Is this about deployment or operations? → `docs/deployment/`
9. Is this a guide, tutorial, or how-to? → `docs/guides/`
10. If none of the above apply → `docs/general/`

**When in doubt:** Use `docs/general/` as the fallback category for any markdown file that doesn't clearly fit into another category.

## Examples

**Example 1: Creating a development plan**
```
User request: "Create a development plan for the new authentication feature"
Filename: DevelopmentPlan.md
Analysis: Contains "Plan" keyword → matches "Development Plans" category
Target: docs/plan/DevelopmentPlan.md
Action: mkdir -p docs/plan && write file to docs/plan/DevelopmentPlan.md
```

**Example 2: Creating a code review**
```
User request: "Document the code review findings"
Filename: CodeReview.md
Analysis: Contains "Review" keyword → matches "Code Reviews" category
Target: docs/review/CodeReview.md
Action: mkdir -p docs/review && write file to docs/review/CodeReview.md
```

**Example 3: Creating API documentation**
```
User request: "Document the REST API endpoints"
Filename: RestAPIEndpoints.md
Analysis: Contains "API" keyword → matches "API Documentation" category
Target: docs/api/RestAPIEndpoints.md
Action: mkdir -p docs/api && write file to docs/api/RestAPIEndpoints.md
```

**Example 4: Creating an ambiguous document**
```
User request: "Create a document about our deployment strategy"
Filename: DeploymentStrategy.md
Analysis: Contains "Deployment" keyword → matches "Deployment" category
Target: docs/deployment/DeploymentStrategy.md
Action: mkdir -p docs/deployment && write file to docs/deployment/DeploymentStrategy.md
```

## Important Notes

- **Always** check if the docs directory and appropriate subdirectory exist before writing
- **Never** write markdown files to the project root or arbitrary locations (except for standard files like README.md, CHANGELOG.md in root)
- **Always** preserve the original filename when moving to the categorized location
- If uncertain about categorization, default to `docs/general/` rather than making an incorrect guess
- This organization applies to **new** markdown files only; do not automatically move existing markdown files without explicit user request
