---
name: onboard
description: Onboard new developers to any codebase. Discovers setup instructions, explores architecture, and suggests features to investigate. Use when a user is new to a repository, asks how to get started, or wants an overview of the codebase.
---

# Developer Onboarding

Welcome new developers to this codebase. This skill helps you:
1. Discover and run setup instructions
2. Understand the technical architecture
3. Explore key features using Plan mode

## Step 1: Discover Project Setup

Search for setup documentation in the repository. You can use the code-explorer subagent here. 

1. **Find documentation files** - Look for:
   - `README.md` or `README` at the root
   - `CONTRIBUTING.md` or `contribute/` directory
   - `docs/`, `documentation/`, or `doc/` directories
   - `DEVELOPMENT.md`, `SETUP.md`, or `GETTING_STARTED.md`
   - `.github/CONTRIBUTING.md`

2. **Identify the tech stack** by examining:
   - `package.json` (Node.js/JavaScript/TypeScript)
   - `go.mod` or `go.sum` (Go)
   - `requirements.txt`, `pyproject.toml`, `setup.py` (Python)
   - `Cargo.toml` (Rust)
   - `pom.xml` or `build.gradle` (Java)
   - `Gemfile` (Ruby)
   - `composer.json` (PHP)
   - `Makefile` (common build automation)
   - `docker-compose.yml` or `Dockerfile` (containerized setup)

3. **Extract prerequisites** from documentation:
   - Required language versions
   - Required tools and dependencies
   - Environment variables needed
   - Database or service dependencies

4. **Present setup steps** to the user in order:
   - Install dependencies
   - Configure environment
   - Build the project
   - Run the application
   - Access the running application (URL, ports)

## Step 2: Explore Architecture

Analyze the codebase structure and present an architecture overview. You can use the code-explorer subagent here. 


1. **List top-level directories** to understand project organization

2. **Identify architectural patterns**:
   - Frontend/backend separation
   - Monorepo vs single project
   - Plugin or module systems
   - API structure (REST, GraphQL, gRPC)

3. **Create a directory summary table** like:

   | Directory | Purpose |
   |-----------|---------|
   | `src/` | Main source code |
   | `tests/` | Test suites |
   | `docs/` | Documentation |

4. **Generate a Mermaid diagram** if the architecture is complex enough to warrant one. Use this template and adapt based on what you discover:

   ```mermaid
   graph TB
       subgraph "Layer 1"
           Component1[component-name/]
           Component2[component-name/]
       end

       subgraph "Layer 2"
           Component3[component-name/]
           Component4[component-name/]
       end

       Component1 --> Component3
       Component2 --> Component4
   ```

5. **Highlight key architectural decisions** found in:
   - `ARCHITECTURE.md` or similar docs
   - Code comments in main entry points
   - Configuration files that reveal patterns

## Step 3: Suggest Features to Explore

Based on the codebase analysis, suggest 3-5 key areas for the developer to explore.

For each feature area:

1. **Identify the feature** and its purpose
2. **Locate key files** or directories
3. **Suggest a Plan mode prompt** for deeper exploration:

   ```
   /plan How does [feature name] work in this codebase?
   ```

### Example Exploration Areas

Depending on the project type, suggest exploring:

- **Web Applications**: Authentication, API routes, database models, UI components
- **Libraries**: Core APIs, plugin systems, configuration
- **CLI Tools**: Command parsing, subcommands, output formatting
- **Services**: API endpoints, middleware, data layer, background jobs

## Step 4: Explain Plan Mode

When exploring complex features, recommend **Plan mode** to:

1. Understand existing implementations before modifying
2. Discuss trade-offs between different approaches
3. Get architectural guidance for new features

Start Plan mode by typing `/plan` followed by a question:

```
/plan I want to add a new [feature]. What's the recommended approach?
```

Plan mode is read-only and helps design before coding.

## Step 5: Provide Development Tips

Discover and share common development workflows:

- **Testing**: Find test commands in `package.json`, `Makefile`, or documentation
- **Linting**: Identify linter configuration (`.eslintrc`, `.golangci.yml`, etc.)
- **Hot reload**: Note if dev servers support hot reloading
- **Configuration**: Point to configuration files and how to override settings

## Step 6: Run the app 

- Run the app in the internal Cursor browser 

## Additional Resources

Search for and link to:

- Contributing guidelines
- Architecture documentation
- Style guides
- API documentation
- Community channels (Discord, Slack, forums)

---

## Agent Instructions

When this skill is invoked:

1. **Start by exploring** - Use file listing and reading tools to discover the repository structure and documentation
2. **Adapt to the project** - Don't assume any specific tech stack; discover it
3. **Be concise** - Summarize setup steps clearly without overwhelming detail
4. **Use Plan mode suggestions** - Tailor exploration prompts to the actual features found in this codebase
5. **Handle missing docs gracefully** - If setup documentation is sparse, infer setup steps from build files and provide best-effort guidance
