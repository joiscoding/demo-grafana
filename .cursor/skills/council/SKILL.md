---
name: council
description: Spawn multiple agents to deeply explore a codebase area before acting. Use when the user invokes /council, asks for a "council" exploration, or wants multi-agent investigation of architecture, features, errors, or code patterns.
---

# Council — Multi-Agent Codebase Exploration

Spawn multiple agents to deeply explore a codebase area before acting. Use when the user wants thorough, multi-perspective investigation of a topic, feature, or problem.

**TIP**: Use params like `n=10` to control agent count (e.g., `n=5`, `n=15`). Default is 10 if not specified.

## Step 1: Gather General Information

Dig around the codebase in the given area of interest. Gather:

- **Keywords**: Relevant terms, class names, file paths, and patterns
- **Architecture overview**: How the area fits into the larger system
- **Key locations**: Directories, modules, or entry points to explore further

Use semantic search, grep, and file exploration. Do not spawn task agents yet.

## Step 2: Spawn Task Agents

Spawn **n** task agents (default `n=10` unless the user specifies otherwise, e.g., `n=5` or `n=15`) to dig deeper into the codebase in that area of interest.

**Agent count**: Parse the user's message for `n=<number>`. If absent, use 10.

**Variance**: Vary the exploration focus across agents. Include:

- Some agents with **focused** queries (specific APIs, flows, or components)
- Some agents with **broader** queries (patterns, integrations, dependencies)
- At least one or two **out-of-the-box** angles (edge cases, error handling, tests, config)

**Subagent type**: Prefer `explore` for codebase exploration. Use `generalPurpose` when the task needs research or multi-step reasoning. Use `code-explorer` when tracing architecture or data flow.

**Launch in parallel**: Spawn all task agents in a single batch when their tasks are independent. Do not wait for one to finish before starting the next.

**Task description**: Give each agent a clear, specific prompt that includes:
- The area of interest from the user
- Keywords and paths from Step 1
- The specific angle or question for that agent

## Step 3: Synthesize and Act

Once the task agents complete:

1. **Synthesize** their findings into a coherent picture
2. **Fulfill the user's request** using the gathered information
3. **If the user is in Plan mode**: Use the information to create the plan

Do not repeat raw agent output. Summarize, reconcile conflicts, and present actionable insights.

## Example Prompts

- `use /council n=15, how does authentication work?`
- `find all places we use InstancedGeometry /council n=5`
- `getting this error, use /council and investigate`
- `/council n=8 explore the alerting pipeline`
