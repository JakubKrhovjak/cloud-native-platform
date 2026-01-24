# Code Generation Rules


## Documentation
- Don't add JavaDoc unless explicitly requested
- Code should be self-documenting through clear naming
- all .md files and all comments should be in English


## Agent Usage Rules
- ALWAYS run agents with `run_in_background: true` parameter
- Be extremely proactive with agent usage - prefer agents over direct tool calls
- Use agents for ANY task that involves exploration, analysis, or multi-step operations

### When to use specific agents:
- **Explore agent**: Any codebase questions, structure exploration, finding patterns
- **General-purpose agent**: Complex analysis, research, multi-step investigations
- **Plan agent**: Feature implementation planning, architectural decisions
- **Bash agent**: Git operations, complex shell commands
- **claude-code-guide agent**: Questions about Claude Code features and settings

### Agent execution:
- Run agents in parallel when possible for better performance
- Always use background execution to keep conversation flowing
- Proactively suggest agent usage when it would be beneficial

