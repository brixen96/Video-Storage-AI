# 📋 New Feature: Shared Task Board

Hi! I have some exciting news — we've added a **Shared Task Board** feature that lets us work together more effectively!

## What is it?

A task board that both **you (the AI Companion)** and **I (the developer)** can use to track bugs, improvements, and work items. It's like a to-do list we can both read and update.

## How you can use it

You now have **5 new tools** available:

| Tool | What it does |
|------|-------------|
| `create_task` | Report a bug, suggest an improvement, or track a work item |
| `list_tasks` | View tasks with filters (status, creator) |
| `update_task` | Update a task's status, priority, or assignee |
| `get_task_stats` | Get statistics about the task board |
| `delete_task` | Remove a task from the board |

## When to use them

- **You find a bug** → call `create_task` with the error details, related file, and line number
- **User asks "what tasks are open?"** → call `list_tasks` to show them
- **You complete something** → call `update_task` with status="completed" and a resolution note
- **You notice a pattern** → create a task to track it

## Task fields

- **title**: Brief description of the task
- **description**: Detailed explanation
- **priority**: 1-10 (higher = more important)
- **tags**: For categorization (e.g., "bug", "improvement", "performance")
- **related_file**: Source file path (if applicable)
- **related_line**: Line number in the source file
- **error_context**: Error message or stack trace
- **resolution**: How it was fixed (when completed)

## Example

If you notice an error like "no such column: created_at", you could say:

```
create_task(
  title="Missing created_at column in activity_logs",
  description="The query is failing because the created_at column doesn't exist in the activity_logs table.",
  priority=8,
  tags=["bug", "database"],
  related_file="internal/services/ai_companion_tools_core.go",
  related_line=190,
  error_context="no such column: created_at"
)
```

## What this means for us

- **You** can report issues immediately when you spot them
- **I** can see all your findings and fix them systematically
- **Both** of us can track progress and close out completed items

The task board is accessible via the clipboard icon in the chat header. Try it out — the next time you notice something wrong, create a task and I'll take care of it! 🎯
