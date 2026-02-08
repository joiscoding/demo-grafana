# Shopify Demo Script

**Duration:** ~30 minutes (+ 10 min colleague browser demo)
**Audience:** Engineers, engineering-adjacent roles, engineering leaders
**Codebase:** Grafana (7M+ lines, Go backend + React frontend)
**Repo:** fieldsphere/grafana

---

## Demo Arc Overview

| Act | Section | Time | What Happens |
|---|---|---|---|
| 1 | Ask Mode | ~4 min | Codebase orientation, semantic search |
| 2 | Skills & Subagents | ~3 min | Team knowledge + fast research |
| 3 | Plan Mode -- Single Issue | ~8 min | Logging fix: plan, align, execute + model selection |
| 4 | Plan Mode -- Parallel Agents | ~5 min | SQL error handling: kick off parallel agents |
| 5 | Council of Agents Review | ~3 min | Kick off multi-agent review of logging changes |
| -- | Buffer / Q&A | ~3 min | Breathe, take questions |
| -- | _Colleague browser demo_ | _~10 min_ | _All agents finish in background_ |
| -- | Show results | ~4 min | Council review + parallel agent output walkthrough |

---

## Act 1: Ask Mode (~4 min)

**Goal:** Show Cursor understanding a massive codebase instantly. Relatable "day 1 at a new codebase" scenario. Land the narrative: what used to take days now takes minutes.

### Verbal Setup (~60 seconds, before touching Cursor)

> One of the biggest challenges in a large codebase is just understanding what's going on. At 7 million lines of code, no single person knows the whole system. When a new engineer joins, or when you're working in an unfamiliar part of the codebase, you need to build context fast.
>
> Think about what that typically looks like. You search for "dashboard" in the codebase and get 10,000 results. You ask a senior engineer to walk you through the architecture -- and they're busy. You check the internal docs and they're either outdated or don't cover the specific flow you need. It can take hours or days to piece together how a request actually flows through the system.
>
> Cursor changes this with semantic search. Unlike grep, which finds exact text matches, semantic search understands meaning. You can ask "where do we handle authentication?" and it finds the right code even if the word "authentication" doesn't appear anywhere. Cursor trained their own embedding model specifically on how coding agents explore codebases -- and it improves accuracy by 12-23% depending on the model, with the biggest gains in large codebases like this one.
>
> Let me show you. I'm going to open Ask Mode -- it's read-only, so it explores the codebase but doesn't change anything.

### Actions in Cursor

**1. Open Ask Mode** (dropdown or Cmd+Period to toggle modes)

**2. Ask a broad question:**

> How is this codebase structured? What are the main architectural layers?

The agent maps the full repo -- Go backend in `pkg/`, React frontend in `public/`, shared packages in `packages/`, etc. The audience sees semantic search and file exploration building a map of a massive repo in seconds.

**3. Follow up with something specific:**

> How does the dashboard API work for **reading a dashboard by UID**?
>
> Please trace the request end-to-end for the legacy HTTP route:
> - GET `/api/dashboards/uid/:uid`
>
> Cite the exact file paths + function names for the frontend entrypoint, frontend HTTP client, backend handler, service layer, and storage/DB layer.

This is the "wow" moment -- Cursor traces a request through six layers across two programming languages: React component, Dashboard API wrapper, HTTP client, Go HTTP handler, service layer, database. It shows depth, not just breadth.

**What to point out in the output:**
- The agent didn't just list files -- it understood the architecture. It traced how data flows down through all six layers and back up.
- It cited specific file paths and function names (`dashboard_api.ts`, `backend_srv.ts`, `pkg/api/dashboard.go`, `GetDashboard()`, etc.)
- It even surfaced that Grafana is actively migrating from a legacy REST API to a Kubernetes-style API -- an ongoing architectural migration that a new engineer might not discover for weeks.

**If the answer comes back too broad**, follow up with:

> Same question, but do NOT cover saving/updating dashboards. Only cover the read path for GET `/api/dashboards/uid/:uid`. End with "Open these 3 files first".

### Takeaway Line (say this after the output appears)

> What you just saw took about 90 seconds. For a new engineer, this is typically a days-long process -- reading docs, asking colleagues, tracing through code manually. And the agent didn't just list files. It traced a request across six layers, two programming languages, and even flagged an ongoing architectural migration. That's not search -- that's comprehension. And it's built on semantic search, which is purpose-built for understanding large codebases.

### Transition to Act 2

> So Ask Mode gave us a solid understanding of this codebase in under two minutes. But notice I was typing those questions from scratch. What if your team had already encoded their common questions and workflows into something reusable?

---

## Act 2: Skills & Subagents (~3 min)

**Goal:** Introduce skills as team knowledge primitives. Light intro to subagents.

### Actions in Cursor

**1. Open the `.cursor/skills/` folder** in the file explorer sidebar. Show the skill names.

### What You Say

> Skills are one of the most powerful customization primitives in Cursor. Think of them as runbooks that the AI actually follows. Instead of tribal knowledge living in someone's head or a Confluence page nobody reads, you encode it as a skill -- and Cursor automatically picks it up when it's relevant.
>
> For example, our team has an issue-tracker skill. It tells Cursor how to pull issues from our GitHub repo, which filters to apply, what format to use. Any engineer on the team gets this behavior automatically -- no setup, no configuration.

### Actions in Cursor

**2. Trigger the skill** -- switch to Agent Mode and say:

> Show me the open issues on our backlog

Cursor recognizes this maps to the issue-tracker skill, runs the `gh` CLI, and pulls back the list of issues. The audience sees: Cursor matched intent to skill, followed team conventions, and returned structured results.

### Bridge to Subagents

> Skills encode _what your team knows_. But Cursor also has built-in intelligence for _how to explore code_. When the agent needs to research something in the codebase, it can spawn lightweight sub-agents -- specialized workers that search fast using smaller, cheaper models. You saw this in action during Ask Mode -- the agent was dispatching sub-agents to explore different parts of the codebase in parallel.
>
> This is the key insight: you don't need one massive model doing everything. You orchestrate multiple models -- fast ones for research, powerful ones for reasoning and planning. We'll see more of this in a minute.

### Transition to Act 3

> Now let's put this all together. I identified issue #12 from that backlog -- the structured logging inconsistency. I'm going to open a fresh chat in Plan Mode and hand the agent that issue directly.

---

## Act 3: Plan Mode -- Logging Issue (#12) (~8 min)

**Goal:** Show the full Plan Mode workflow. Demonstrate issue-to-fix continuity from Act 2. Introduce model selection (plan with a reasoning model, execute with a faster model).

**Issue:** #12 -- "Inconsistent logging. Update all logging to structured logging."

### Verbal Setup

> I'm opening a new chat here, and that's intentional. I triaged the backlog in one session, now I'm starting a focused session to fix a specific issue. That's how this works in practice -- you don't need to re-explain the problem to the agent. I'm going to reference the issue directly and let the agent pull the context it needs.
>
> And I'm starting in Plan Mode. When you're working with AI coding agents, there's a common frustration: you describe what you want, the agent goes off and starts making changes, and then you realize it misunderstood your intent. Now you're spending time undoing work or course correcting.
>
> Plan Mode solves this by creating a checkpoint between your prompt and the agent's execution. Before the agent touches any code, it shows you exactly what it understood and what it plans to do. You review the plan and only then does execution begin. Misunderstandings get caught early, not after the agent has already edited 20 files.
>
> It also saves tokens. When an agent has a clear, reviewed plan, it executes more efficiently. Fewer exploration loops, less refactoring -- that translates directly into lower costs and faster completions.

### Actions in Cursor

1. **Open a new chat** (Cmd+N)
2. **Switch to Plan Mode** (dropdown or Cmd+Period)
3. **Paste the prompt:**

> Fix issue #12. The logging is inconsistent across the codebase -- some places use structured logging with key-value pairs, which is what we want, but others use fmt.Sprintf or format strings that Grafana's logging library doesn't interpret. Find all instances and fix them to use proper structured logging.

3. **While the agent plans (~30-60 sec)** -- narrate or engage audience (see talk tracks below)
4. **Review the plan** -- the agent should identify two problem types:
   - Type A: `fmt.Sprintf` wrapping log messages (works but inconsistent)
   - Type B: Format strings without Sprintf (actively broken)
5. **Switch model** (reasoning model to a faster execution model) for execution
6. **Execute** -- watch it work through the files
7. **Show the diff** -- before/after of a broken format string becoming clean structured logging

### Talk Tracks While Agent Plans

Pick one if you need to fill time:

- **Narrate:** "Watch what's happening -- the agent is searching across the codebase, reading files, categorizing what it finds. It's not just doing find-and-replace. It's understanding the intent of each logging call to decide how to fix it."
- **Contextualize:** "In a codebase this size, an engineer might spend half a day just finding all the instances. The agent is doing that discovery phase right now."
- **Engage audience:** "While this runs -- has anyone dealt with a logging standardization effort before? How long did it take?"

### Talk Tracks While Agent Executes

Pick one if you need to fill time:

- **Explain model switch:** "Notice I switched models for execution. The plan was the hard part -- understanding the codebase, categorizing instances, deciding the approach. That's reasoning work. Execution is well-defined now: the plan says exactly what to do in each file. So I hand it to a faster model. Faster completion, lower cost, same quality."
- **Connect to workflow:** "This is the pattern we see working well in large orgs: plan with a powerful model, execute with a fast one. Your senior architect designs the approach, your team executes it. Same idea, but with models."
- **Engage audience:** "Quick question -- when you think about AI coding costs at scale, what's the bigger concern: token cost or engineer wait time?"

### Audience Q&A (if questions come up)

| Question | Answer |
|---|---|
| What if the plan is wrong? | That's exactly why Plan Mode exists. You review before execution. If something's off, you refine the plan -- no code has been touched yet. |
| How does it know which files to change? | The agent uses semantic search and grep together. Semantic search finds code by meaning, grep finds exact patterns. The combination is what makes it thorough. |
| Could it miss some instances? | It could. That's why the plan shows you exactly what it found. You can verify before executing, and run it again to catch stragglers. |
| Why not just use a regex? | Some of these could be regex'd, but the two pattern types need different fixes. The agent understands the difference. |

### Transition to Act 4

> So that was one plan, one agent, one execution. But what happens when a task is big enough that it can be split across multiple agents working simultaneously?

---

## Act 4: Plan Mode -- Parallel Agents (#10) (~5 min)

**Goal:** Show parallel agent execution. Reinforce the model orchestration narrative.

**Issue:** #10 -- "SQL sources missing error handling."

### Verbal Setup

> Here's issue #10 -- SQL sources are silently swallowing errors. The fix needs to happen in two areas: the SQL datasources (MySQL, PostgreSQL, MSSQL all have the same pattern) and Elasticsearch. These are completely independent -- different folders, different files, no shared code. That's the perfect shape for parallel agents.

### Actions in Cursor

1. **Switch to Plan Mode**
2. **Paste the prompt:**

> Fix issue #10: SQL sources missing error handling. The MySQL, PostgreSQL, and MSSQL datasource macro functions have TODO comments where error handling was never implemented -- errors from regexp.Compile are silently ignored. Also, Elasticsearch responses return generic "unexpected status code" errors instead of extracting the actual error message. Create a plan that splits this into parallel work units -- one for SQL datasources and one for Elasticsearch.

3. **Review the plan** -- should show two parallel tracks:
   - Track 1: SQL datasources (MySQL, PostgreSQL, MSSQL in `pkg/tsdb/mysql/`, `pkg/tsdb/grafana-postgresql-datasource/`, `pkg/tsdb/mssql/`)
   - Track 2: Elasticsearch (`pkg/tsdb/elasticsearch/`)
4. **Kick off parallel agents** -- launch separate agents for each track
5. **Don't wait for them to finish**

### What You Say as You Kick Them Off

> Now I'm going to launch separate agents for each track. Each one gets its own slice of the plan and works independently -- two agents, two work units, running at the same time.
>
> This is where model orchestration really matters. I planned with a deep reasoning model that understood the architecture and decided how to split the work. Now each agent executes with a fast model on a well-scoped task. Plan smart, execute fast -- multiplied across parallel workers.

### Then Transition

> These will keep running in the background. But before we hand off, I want to kick off one more thing.

### Talk Tracks (use if needed during planning)

- **Why parallel matters:** "In a large org, this is the shape of real work. You have a tech debt issue that touches five services. Traditionally one engineer works through them sequentially, or you split across a team and coordinate via PRs. With parallel agents, one person kicks off multiple focused workers and reviews the output."
- **Connect to Shopify:** "Think about a payment service refactor -- updating error handling across Stripe, PayPal, Shop Pay. All independent integrations. Same pattern: one plan, parallel execution."
- **Engage audience:** "How do you all handle cross-cutting changes like this today? One engineer or split across the team?"

### Audience Q&A

| Question | Answer |
|---|---|
| Can the agents see each other's work? | No, each works independently on its own scope. The plan is what coordinates them. |
| What if one finishes before the others? | That's fine. They're independent. You can review the finished one while others are still running. |
| How many agents can you run at once? | Several. The practical limit is usually how you want to review the output. |
| What models are available? | Cursor works with models across labs -- OpenAI, Google, Anthropic. You pick the best model for the job. |

---

## Act 5: Council of Agents Review -- Kick Off (~3 min)

**Goal:** Show multi-agent review pattern. Kick off the council before the browser demo so everything runs in background.

### Verbal Setup

> Before we hand off, I want to kick off one more thing. In the logging change earlier, we used one model to do the deep reasoning and planning, then switched to a different model for fast execution. That's already model orchestration on a single task.
>
> But here's where it gets really interesting. Instead of just asking one model to review, I'm going to use a council approach -- spawning custom sub-agents, each pinned to a different model from a different lab, to review these changes from different angles simultaneously.
>
> There's a key insight here: models tend to like code that looks like what they generate. Claude wrote this code. If I ask Claude to review it, it's like a writer editing their own work -- blind spots persist. So instead, I've set up four reviewer agents pinned to OpenAI and Google models. Different training data, different reasoning patterns, different blind spots.

### Actions in Cursor

1. **Briefly show the `.cursor/agents/` folder** -- open one reviewer file to show the YAML frontmatter with the pinned model. Point out that each has a different model.

> These are custom sub-agents. Each one has a specific review mandate and a specific model pinned to it. The correctness reviewer runs on GPT-5.2 from OpenAI. The edge case reviewer runs on Gemini 3 Pro from Google. They're all read-only -- they can analyze but can't modify code. And because they're files in the repo, every engineer on the team gets the same review council.

2. **Ask the agent to run the council review:**

> Council review the structured logging changes we just made in pkg/api/ and pkg/services/

3. **Watch it fan out** -- four sub-agents launch in parallel, each pinned to its own model
4. **Don't wait for results** -- transition to the browser demo

### What You Say as They Launch

> Watch -- four sub-agents just launched, all running at the same time, each with a different model. The correctness reviewer on GPT-5.2, the completeness reviewer on GPT-5.2 Codex, the edge case reviewer on Gemini 3 Pro, and the style reviewer checking codebase conventions.
>
> This is the council of agents pattern. Claude Opus wrote the code. Now OpenAI and Google models are reviewing it. These will finish in the background along with our parallel agents from earlier. Let's hand off to [colleague] for the browser demo, and we'll come back to the results.

---

## Colleague Browser Demo (~10 min)

_Hand off to colleague. Parallel agents and council review all running in background._

---

## Show Results (~4 min)

**Goal:** Walk through completed output from both the council review and parallel agents. Close with the strongest orchestration narrative.

### Council Review Results

> Let's check on our council review first.

Walk through the synthesized review. Either outcome is a win:

- **If it found something:** "See -- the sub-agents caught something the original execution missed. Different models, different blind spots. That's the whole point of cross-lab review."
- **If it validated:** "Four agents, three models, two labs -- all agree the changes are solid. That's a much stronger signal than one model checking its own work."

**What to point out:**

> When models from different labs converge on the same conclusion -- that's not one model checking its own work. That's independent verification. That's real confidence.

### Parallel Agent Output

> Now let me check on those parallel agents we kicked off earlier.

Quick walkthrough of the completed SQL error handling work across the SQL datasources and Elasticsearch.

### Closing Line

> Plan with a reasoning model, execute with a fast model, review with a council of agents from different labs. Claude wrote the code, OpenAI and Google reviewed it. You go from writing code to orchestrating a team of AI specialists -- each with a pinned model, a clear mandate, and a genuinely independent perspective. In a 7 million line codebase, that's not just faster -- it's how you maintain quality at scale.

---
---

# Concepts Reference

A reference guide for the key technical concepts in this demo. Written for someone who understands general developer concepts but isn't a developer themselves.

---

## Cursor Modes

### Ask Mode

**What it is:** A read-only mode where the agent explores and answers questions about the codebase but can't edit any files.

**When to use it:** Onboarding to a new codebase, understanding how something works, code review, or any time you want information without changes.

**How to switch:** Dropdown in the chat panel, or Cmd+Period to toggle between modes.

### Plan Mode

**What it is:** A two-phase mode. Phase 1: the agent reads the codebase and creates a plan (read-only). Phase 2: you approve the plan and the agent executes it (makes changes).

**Why it matters:** It creates a checkpoint between your intent and the agent's actions. You catch misunderstandings before code is edited, not after. It also saves tokens -- a clear plan means fewer wasted exploration loops during execution.

**How to switch:** Same dropdown or Cmd+Period.

### Agent Mode

**What it is:** The default mode. The agent reads, writes, and executes code directly based on your prompt. No explicit planning phase.

**When to use it:** Well-understood tasks, small changes, or when you trust the agent to figure it out.

### Debug Mode

**What it is:** A mode optimized for investigating bugs. The agent follows a systematic approach: understand symptoms, trace to root cause, propose a fix.

**When to use it:** Bug reports, error messages, test failures, unexpected behavior.

---

## Semantic Search

**What it is:** A way to search code by meaning, not just exact text. When you ask "where do we handle authentication?" it finds relevant code even if the word "authentication" doesn't appear in the files.

**How it works:** Cursor converts code into numerical representations called embeddings -- think of them as coordinates in a space where similar code ends up near each other. Your question gets converted the same way, and Cursor finds the code whose "meaning" is closest to your question's meaning.

**Why it matters in large codebases:** Grep (exact text search) returns thousands of results in a 7M line codebase. Semantic search understands what you're actually looking for and returns the relevant results. Cursor trained their own embedding model specifically on how coding agents work through tasks -- so it knows what's useful to find, not just what's textually similar.

**Key stat:** 12-23% higher accuracy in answering questions across all frontier models, with the biggest gains in large codebases.

**Comparison to grep:** The agent uses both. Grep is great for exact matches ("find every file that imports this package"). Semantic search is great for conceptual questions ("how does the auth flow work?"). Together they provide thorough coverage.

---

## Skills

**What they are:** Markdown files in `.cursor/skills/` that contain reusable instructions for the AI agent. They encode team-specific knowledge -- which repo to use, how to run things, what conventions to follow.

**How they work:** Each skill has a name, a description, and instructions. Cursor reads the description and automatically matches it to your request. You don't have to say "use the issue-tracker skill" -- you just say "show me issues" and Cursor figures out which skill applies.

**Why they matter for enterprises:** They standardize how AI works across a team. Instead of every engineer getting different results because they prompt differently, skills ensure consistent behavior. Platform teams can create skills for internal tools, CI/CD patterns, coding standards, etc.

**Example:** The issue-tracker skill tells Cursor: "when someone asks about issues, use the `gh` CLI, target the `fieldsphere/grafana` repo, use these specific flags." Every engineer gets the same behavior automatically.

---

## Subagents

**What they are:** Smaller, focused agents that the main agent spawns to do specific work. Think of them as specialists the main agent delegates to.

**The built-in explore subagent:** Cursor has a built-in subagent optimized for codebase research. It uses fast, cheap models to scan and read code quickly. When the main agent needs to understand something, it sends out an explore subagent rather than doing the slow reading itself.

**Analogy:** The main agent is a senior engineer. Subagents are junior researchers who can scan the codebase in parallel while the senior engineer focuses on decisions.

---

## Model Selection and Orchestration

### The Core Idea

Different AI models are good at different things. Instead of using one model for everything, you match the model to the task:

| Task | Best Model Type | Example |
|---|---|---|
| Planning, architecture | Deep reasoning model | GPT (OpenAI) / Gemini (Google) / Claude (Anthropic) |
| Fast code execution | Speed-optimized model | A faster coding model (same or different provider) |
| Codebase research | Lightweight, cheap model | Explore subagent |
| Review | Different lab for independent perspective | GPT (OpenAI), Gemini (Google) |

### Why Multi-Lab Matters

Models from the same lab tend to think in similar patterns. When a model from Lab A reviews code written by a model from Lab A, it's less likely to catch issues -- similar to a writer editing their own work.

When you bring in a model from Lab B, you get genuinely independent review. Different training data, different reasoning patterns, different blind spots. If both agree, your confidence goes way up.

Cursor is model-agnostic -- it works with OpenAI, Google, Anthropic, and others. This is a competitive advantage over tools locked to a single provider.

### The Pattern for This Demo

1. **Plan** with a reasoning model -- deep reasoning to understand the codebase and design the approach
2. **Execute** with a faster model -- fast, efficient execution on well-defined tasks
3. **Research** with explore subagents -- cheap, fast codebase scanning
4. **Review** with a council of agents -- multiple sub-agents from different angles for broad coverage

---

## Parallel Agents

**What they are:** Multiple agents working on separate parts of a codebase simultaneously. Each agent gets its own scoped task and works independently.

**When they work well:** When a task naturally splits into independent parts -- different packages, different services, different files with no overlap. The key requirement is that agents won't step on each other's work.

**How they're coordinated:** The plan defines the boundaries. Each agent gets a specific track (e.g., "fix error handling in the MySQL package"). The plan is the contract that keeps them from conflicting.

**The shift for engineers:** You go from being an executor (writing code yourself) to an orchestrator (defining work, launching agents, reviewing output). One person can drive work that would traditionally require a team.

---

## Council of Agents

**What it is:** A workflow pattern where the main agent spawns multiple sub-agents to research or review a problem from different angles before synthesizing the results.

**How it works:**
1. You give the agent a task (e.g., "review these changes")
2. The agent fans out multiple sub-agents, each with a different focus
3. Sub-agents explore independently and in parallel
4. The main agent synthesizes all findings into one actionable summary

**Why it's powerful:** Single-model review is like having one person QA your code. A council is like having engineering, security, and performance teams all review simultaneously. Broader coverage, faster, and more likely to catch issues.

**The enterprise vision:** Imagine specialized agents -- a Product agent, a Security agent, a Performance agent -- collaborating like a real engineering team. One puts up a PR, another reviews for security concerns, another flags performance regressions. That's the direction this is heading.

---

## Grafana Codebase (Quick Reference)

The codebase you're demoing in. Useful if someone asks about the repo structure.

| Folder | What's In It |
|---|---|
| `pkg/` | Go backend -- API handlers, services, database layer, plugins |
| `public/` | React frontend -- components, pages, state management |
| `packages/` | Shared TypeScript libraries used across the frontend |
| `apps/` | Grafana sub-applications |
| `conf/` | Configuration files |
| `docs/` | Documentation |
| `e2e/` | End-to-end tests |
| `scripts/` | Build and utility scripts |

### The Dashboard API Flow (from Act 1)

When someone opens a dashboard in Grafana (e.g. `GET /api/dashboards/uid/:uid`), six layers are involved:

1. **React UI** (`public/app/features/dashboard/`) -- user navigates to a dashboard, a React component fires
2. **Dashboard API wrapper** (`public/app/features/dashboard/api/dashboard_api.ts`) -- decides which API version to call (Grafana is migrating between two API styles)
3. **BackendSrv HTTP client** (`public/app/core/services/backend_srv.ts`) -- sends the actual HTTP request, handles auth tokens, retries, error handling
4. **Go HTTP handler** (`pkg/api/dashboard.go`) -- receives the request, checks permissions via RBAC, passes it down
5. **Dashboard service** (`pkg/services/dashboards/service/`) -- business logic layer, fetches dashboard data and assembles the response
6. **Database layer** (`pkg/services/dashboards/database/`) -- runs the actual SQL query against the database

The request flows down through all six layers and the response flows back up. Cursor traces this entire flow when you ask about the dashboard API.

**Key architectural detail:** Grafana is actively migrating from their original REST API (`/api/dashboards/...`) to a Kubernetes-style API (`/apis/dashboard.grafana.app/...`). The service layer uses a "dual-write" pattern to keep both old and new storage in sync during the transition. This is a real-world migration pattern -- you rarely rip and replace, you run both in parallel.

---
---

# Presenter Notes: Technical Concepts by Act

Notes on the technical concepts you need to understand for each act. Written at the "explain it to a smart non-engineer" level -- enough to narrate confidently and handle audience questions.

---

## Act 1 Notes: Ask Mode & Dashboard API Flow

### What is Grafana?

Grafana is a monitoring and observability platform. Companies use it to build dashboards that visualize data -- server health, error rates, application performance, business metrics. You connect it to your databases and data sources, and it gives you charts, graphs, and alerts. It's open-source, used by thousands of companies, and the codebase is massive: 7M+ lines of code in two main languages (Go for the backend, TypeScript/React for the frontend).

### The Two Languages

- **Go (backend):** The server-side code that handles requests, enforces permissions, talks to the database. Lives in `pkg/`. Go is a popular language for backend services -- fast, reliable, widely used at companies like Google, Uber, and Shopify.
- **TypeScript/React (frontend):** The browser-side code -- what users see and interact with. Lives in `public/`. React is a UI framework built by Meta (Facebook).

### The Dashboard Request Flow (Simple Version)

**Going down (fetching):**
1. **React UI** -- User clicks a dashboard in the browser, which fires off a request.
2. **HTTP Client** -- The frontend's messenger; it packages the request and sends it to the backend server.
3. **Go Handler** -- The backend's front door; it receives the request and checks "does this user have permission to see this?" (RBAC).
4. **Service Layer** -- The brain; it contains the business logic for how to actually get a dashboard and what extra info to attach.
5. **Storage / Database** -- The source of truth; it runs the query and returns the raw dashboard data.

**Coming back up (responding):**
6. The storage hands data back to the service, the service hands it to the handler, the handler packages it as JSON, the HTTP client receives it, and React renders the dashboard on screen.

### The Dashboard Request Flow (Restaurant Analogy)

When a user opens a dashboard, a request travels down through six layers and the response travels back up. Think of it like a restaurant:

| Layer | Restaurant Analogy | What It Does |
|---|---|---|
| 1. React UI | The customer | User clicks on a dashboard. The browser fires a request. |
| 2. API / HTTP Client | The waiter | Takes the order (request) and carries it to the kitchen (backend). Doesn't cook -- just relays. |
| 3. Go HTTP Handler | The head chef | Receives the order. First checks: "Is this person allowed to order?" (permissions/RBAC). If yes, hands it to the line cook. |
| 4. Service Layer | The line cook | Knows the recipe (business logic). Figures out what data is needed, pulls it together. |
| 5. Storage / Database | The pantry | Where the raw ingredients (data) live. The line cook grabs what they need. |
| 6. Response back up | Food comes out | Storage → Service → Handler → HTTP Client → React UI renders the dashboard on screen. |

**Key point:** This is a READ operation. Nothing is being written. Data flows down to fetch, back up to display.

### Handler vs Service Layer (Common Confusion)

These two are easy to mix up. The simple distinction:

- **Handler** = deals with HTTP. Its job: receive the web request, pull the dashboard UID out of the URL, check permissions, and at the end, package everything into a JSON response. It's the boundary between "the internet" and "our application logic."
- **Service layer** = deals with business logic. Its job: actually fetch the dashboard from storage, figure out what folder it's in, check if it's provisioned, etc. It doesn't know or care about HTTP.

The handler calls the service. The service does the real work. The handler wraps the result in a nice JSON response.

### Key Terms to Know

| Term | Plain English |
|---|---|
| **API** | The contract between frontend and backend. The frontend says "give me dashboard X" via a URL, the backend responds with data. |
| **UID** | A unique identifier for a dashboard -- like a license plate. Every dashboard has one. |
| **RBAC** | Role-Based Access Control. Before serving a dashboard, the system checks: does this user have permission to read it? |
| **Dual-write pattern** | During a migration, you write to both the old and new systems simultaneously so nothing breaks. Lets you switch over gradually instead of a risky all-at-once cutover. |
| **Kubernetes-style API** | A newer, more structured way of organizing APIs that Grafana is migrating toward. You don't need to explain what Kubernetes is -- just that it's a "newer architecture pattern" and the migration is in progress. |

### What Makes Act 1 Impressive (Your Takeaway)

The punchline is **speed + depth**. Cursor didn't just list files -- it traced a request across six layers, two programming languages, cited specific file paths and function names, and even surfaced an ongoing architectural migration that a new engineer might not discover for weeks. That whole exercise took ~90 seconds. For a new engineer, this is normally a days-long process of reading docs, asking colleagues, and tracing code by hand.

### Handling Audience Questions

| Question | Your Answer |
|---|---|
| "Why six layers? Isn't that overkill?" | Each layer has a single job. The handler deals with HTTP, the service deals with logic, the store deals with data. Separation of concerns makes it easier to change one layer without breaking the others. That's standard in enterprise codebases. |
| "What's the dual-write pattern?" | During a migration, you write to both old and new systems at the same time so nothing breaks. You switch over gradually instead of a risky big-bang cutover. |
| "How is this different from grep?" | Grep finds exact text matches -- search for "dashboard" and get 10,000 results. Semantic search understands meaning. You ask "how does the dashboard API work?" and it finds the right code even if those exact words don't appear. Cursor uses both together. |
| "Could a new engineer just read the docs instead?" | They could try, but docs are often outdated or don't cover the specific flow you need. Cursor reads the actual code -- the source of truth -- and traces it live. |
