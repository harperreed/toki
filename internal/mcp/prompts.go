// ABOUTME: MCP prompt definitions and handlers
// ABOUTME: Provides workflow templates and best practices for using toki

package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerPrompts() {
	s.registerPlanProjectPrompt()
	s.registerDailyReviewPrompt()
	s.registerSprintPlanningPrompt()
	s.registerTrackAgentWorkPrompt()
	s.registerCoordinateTasksPrompt()
	s.registerReportStatusPrompt()
}

func (s *Server) registerPlanProjectPrompt() {
	s.mcp.AddPrompt(&mcp.Prompt{
		Name:        "plan-project",
		Description: "Break down a new project into actionable tasks with phases, priorities, and tags. Use when starting work on a new initiative or feature.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "project_name",
				Description: "Name of the project to plan (optional, will be prompted if not provided)",
				Required:    false,
			},
		},
	}, s.handlePlanProject)
}

//nolint:funlen // Prompt handlers contain large template strings
func (s *Server) handlePlanProject(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	projectName := ""
	if req.Params.Arguments != nil {
		projectName = req.Params.Arguments["project_name"]
	}
	if projectName == "" {
		projectName = "[project name]"
	}

	template := fmt.Sprintf(`# Plan Project: %s

## Overview
This workflow helps you break down a new project into manageable, trackable tasks in toki. You'll identify the project scope, organize work into phases, assign priorities, and create tagged todos that can be tracked through completion.

## When to Use
- Starting a new feature, initiative, or body of work
- Need to decompose a large effort into smaller tasks
- Want to organize work across multiple phases or workstreams
- Setting up a new project in toki with initial backlog

## Workflow Steps

### Step 1: Create the Project
Use **add_project** to create a container for all related tasks.

**Example:**
- Call add_project with name="%s"
- Optionally include path to git repository
- Save the project_id for use in subsequent todos

### Step 2: Identify Major Phases
Break the project into 2-5 major phases or workstreams.

**Examples of phases:**
- "setup", "implementation", "testing", "deployment"
- "backend", "frontend", "infrastructure"
- "phase-1", "phase-2", "phase-3"

Each phase will become a tag to organize related todos.

### Step 3: Create High-Level Tasks
For each phase, identify the major tasks needed.

**Use add_todo with:**
- description: Clear, actionable task description
- project_id: The project UUID from Step 1
- priority: "high" for critical path, "medium" for important, "low" for nice-to-have
- tags: Include the phase tag plus any other relevant categories
- notes: Add context, dependencies, or implementation details
- due_date: Set deadlines for time-sensitive tasks (ISO 8601 format)

**Example:**
- add_todo(description="Set up database schema", project_id="...", priority="high", tags=["setup", "database"], notes="Need migrations for users, posts, comments tables")

### Step 4: Review and Prioritize
Use **list_todos** to review all project tasks:
- Check the task breakdown is complete
- Verify priorities reflect actual criticality
- Ensure tags are consistent and useful
- Confirm due dates align with project timeline

**Example:**
- list_todos(project_id="...")
- Review by priority: list_todos(project_id="...", priority="high")
- Review by phase: list_todos(project_id="...", tag="setup")

### Step 5: Identify Dependencies
Add notes to todos that have dependencies on other tasks. Use the description or todo ID of the blocking task.

**Example:**
- update_todo(todo_id="...", notes="Blocked by: 'Set up database schema' (abc123...)")

## Tips and Best Practices
- **Start broad, refine later:** Don't try to identify every task upfront. Create 10-20 high-level tasks, then break them down as you start work.
- **Use consistent tags:** Decide on phase/category tags early and stick to them. This makes filtering much easier.
- **Priority discipline:** Be realistic with "high" priority. If everything is high priority, nothing is.
- **Actionable descriptions:** Write todos as actions: "Implement X", "Write Y", "Deploy Z" - not "X needs implementation".
- **Due dates for milestones:** Set due dates on tasks that block other work or have external deadlines. Don't date everything.
- **Review the plan:** Use list_todos with different filters to verify your breakdown makes sense.

## Example
**Project:** REST API for blog platform

**Step 1:** Create project
- add_project(name="blog-api", path="/home/user/projects/blog-api")
- Result: project_id = "550e8400-..."

**Step 2:** Identify phases
- Tags: "setup", "core", "features", "testing", "deployment"

**Step 3:** Create initial tasks
- add_todo(description="Set up Go project with dependencies", project_id="550e8400-...", priority="high", tags=["setup"], due_date="2025-12-05T00:00:00Z")
- add_todo(description="Design and implement database schema", project_id="550e8400-...", priority="high", tags=["setup", "database"])
- add_todo(description="Implement user authentication endpoints", project_id="550e8400-...", priority="high", tags=["core", "auth"])
- add_todo(description="Implement CRUD for blog posts", project_id="550e8400-...", priority="high", tags=["core", "posts"])
- add_todo(description="Add full-text search for posts", project_id="550e8400-...", priority="medium", tags=["features", "search"])
- add_todo(description="Write integration tests", project_id="550e8400-...", priority="medium", tags=["testing"])
- add_todo(description="Set up CI/CD pipeline", project_id="550e8400-...", priority="medium", tags=["deployment"])

**Step 4:** Review the plan
- list_todos(project_id="550e8400-...") → 7 todos
- list_todos(project_id="550e8400-...", priority="high") → 4 high priority items to tackle first
- list_todos(project_id="550e8400-...", tag="setup") → 2 setup tasks to complete before development

**Ready to start planning?**
1. Create your project with add_project
2. Identify your phases (3-5 tags)
3. Create 10-20 high-level todos with priorities and tags
4. Review with list_todos to verify the breakdown
`, projectName, projectName)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Project planning workflow for: %s", projectName),
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: template},
			},
		},
	}, nil
}

func (s *Server) registerDailyReviewPrompt() {
	s.mcp.AddPrompt(&mcp.Prompt{
		Name:        "daily-review",
		Description: "Daily standup workflow to check overdue items, review priorities, identify blockers, and plan the day's focus.",
		Arguments:   []*mcp.PromptArgument{},
	}, s.handleDailyReview)
}

//nolint:funlen // Prompt handlers contain large template strings
func (s *Server) handleDailyReview(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	template := `# Daily Review

## Overview
A daily standup workflow to assess your current workload, identify urgent items, clear blockers, and decide on today's focus. This 5-10 minute review helps you start each day with clarity and intention.

## When to Use
- Start of each workday
- After completing a major task and deciding what's next
- When feeling overwhelmed and need to regain focus
- Monday morning to plan the week

## Workflow Steps

### Step 1: Check Overdue Items
Overdue todos are tasks that missed their deadline and need immediate attention.

**Use toki://todos/overdue resource or:**
- list_todos(overdue=true)

**Actions to take:**
- Review each overdue todo
- For each item, decide:
  - **Do now:** Still critical, make it today's top priority
  - **Reschedule:** Update due_date to realistic timeline
  - **Cancel:** No longer relevant, delete_todo or mark_done
  - **Delegate:** Add notes about who will handle it

**Example:**
- See "Write API documentation" overdue by 3 days
- Still important but blocked waiting for API finalization
- update_todo(todo_id="...", due_date="2025-12-10T00:00:00Z", notes="Blocked waiting for API v2 finalization")

### Step 2: Review High-Priority Pending Items
Focus on what matters most.

**Use toki://todos/high-priority resource or:**
- list_todos(done=false, priority="high")

**Actions to take:**
- Verify these are actually high priority (not everything can be urgent)
- Check if any high-priority items are actually blockers for other work
- Identify 1-3 high-priority items to focus on today

**Example:**
- 8 high-priority todos found
- 3 are actually "medium" - update_todo to fix priority
- 2 are blocked waiting on others - add notes
- 3 are genuinely critical - these become today's focus

### Step 3: Identify Blockers
Blockers are long-pending todos that might be stuck.

**Find todos created >7 days ago that are still pending:**
- list_todos(done=false)
- Manually check created_at dates for old items

**For each blocker, diagnose:**
- **Unclear requirements:** Add notes with questions, reach out to clarify
- **Waiting on others:** Add notes about who/what you're waiting for
- **Too large:** Break into smaller todos
- **Lost motivation:** Either delete it or set a due date to force action

**Example:**
- "Refactor user service" has been pending for 14 days
- Diagnosis: Too large and vague
- Action: Break into 3 smaller todos, delete the original

### Step 4: Plan Today's Focus
Decide on 1-3 todos to complete today.

**Selection criteria:**
- High priority items from Step 2
- Overdue items from Step 1 that must be done now
- Unblocked items that unlock other work
- Items that align with weekly/sprint goals

**Use toki://stats resource to see overall workload:**
- Check pending count - is it growing or shrinking?
- Check oldest pending - is anything rotting?

**Document your plan:**
- Add due_date of today to your chosen focus items
- Or simply keep them in mind and check them off as you go

**Example focus for today:**
1. "Implement user authentication endpoints" (high priority, unblocks frontend work)
2. "Fix bug in payment processing" (overdue, customer-facing)
3. "Write integration tests for auth" (high priority, pairs with #1)

### Step 5: Clean Up
Quick maintenance to keep toki useful.

**Actions:**
- Mark any todos done that you forgot to update yesterday
- Delete todos that are no longer relevant
- Add new todos for work you discovered yesterday
- Update priorities if they've changed

**Example:**
- Completed "Set up CI pipeline" yesterday, forgot to mark it
- mark_done(todo_id="...")

## Tips and Best Practices
- **Consistency over perfection:** Do this every day, even if brief. 5 minutes is better than skipping.
- **Overdue doesn't mean failure:** Deadlines slip. The important thing is acknowledging it and adjusting.
- **Limit daily focus:** Committing to 1-3 items is realistic. 10 items guarantees failure.
- **Use resources for speed:** toki://todos/overdue and toki://todos/high-priority are faster than manual filtering.
- **Check stats weekly:** Use toki://stats every Monday to see trends in your backlog.
- **Be honest about blockers:** If something has been pending for weeks, it's blocked. Acknowledge it.

## Example Daily Review

**Step 1: Overdue**
- list_todos(overdue=true) → 2 items
- "Write API docs" → reschedule to next week
- "Fix login bug" → critical, do today

**Step 2: High Priority**
- list_todos(done=false, priority="high") → 5 items
- Review list, 2 are actually medium priority
- 3 genuine high priority items to consider

**Step 3: Blockers**
- list_todos(done=false) → review created_at dates
- "Refactor auth service" pending for 12 days
- Too vague, break into smaller tasks

**Step 4: Today's Focus**
1. "Fix login bug" (overdue, customer-facing)
2. "Implement password reset endpoint" (high priority, clear requirements)
3. "Review PR for new feature" (medium priority, helps team)

**Step 5: Clean Up**
- mark_done for 2 todos completed yesterday
- Delete 1 todo that's no longer relevant

**Ready to start your day?**
1. Check toki://todos/overdue
2. Review toki://todos/high-priority
3. Look for blockers in pending todos
4. Choose 1-3 focus items for today
5. Quick cleanup of yesterday's work
`

	return &mcp.GetPromptResult{
		Description: "Daily standup and planning workflow",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: template},
			},
		},
	}, nil
}

func (s *Server) registerSprintPlanningPrompt() {
	s.mcp.AddPrompt(&mcp.Prompt{
		Name:        "sprint-planning",
		Description: "Organize work into a focused iteration (sprint) by reviewing backlog, grouping by priority/tags, and setting sprint goals.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "sprint_duration",
				Description: "Duration of the sprint (optional, default: 2 weeks)",
				Required:    false,
			},
		},
	}, s.handleSprintPlanning)
}

//nolint:funlen // Prompt handlers contain large template strings
func (s *Server) handleSprintPlanning(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	sprintDuration := ""
	if req.Params.Arguments != nil {
		sprintDuration = req.Params.Arguments["sprint_duration"]
	}
	if sprintDuration == "" {
		sprintDuration = "2 weeks"
	}

	template := fmt.Sprintf(`# Sprint Planning

## Overview
Organize your backlog of pending todos into a focused iteration (sprint) with clear goals, realistic scope, and prioritized work. This workflow helps you commit to what you can actually accomplish in %s rather than carrying an overwhelming backlog.

## When to Use
- Starting a new sprint or iteration cycle
- Need to focus scattered work into clear goals
- Backlog has grown large and needs organization
- Team wants to align on shared objectives

## Workflow Steps

### Step 1: Review Full Backlog
Get a complete picture of all pending work.

**Use toki://todos/pending resource or:**
- list_todos(done=false)

**Actions:**
- Note the total pending count
- Scan for todos that are outdated or no longer relevant
- Delete or mark_done any todos that don't need to be tracked

**Example:**
- list_todos(done=false) → 47 pending todos
- Find 5 that are outdated, delete_todo for each
- Now have 42 todos to consider for sprint

### Step 2: Group by Priority
Understand the distribution of work urgency.

**Check each priority level:**
- list_todos(done=false, priority="high")
- list_todos(done=false, priority="medium")
- list_todos(done=false, priority="low")

**Actions:**
- Verify priorities are accurate (not everything can be high)
- High priority should be < 30%% of backlog
- Adjust priorities where needed with update_todo

**Example:**
- High: 18 todos (too many!)
- Medium: 20 todos
- Low: 4 todos
- Action: Demote 8 "high" to "medium" to reflect reality

### Step 3: Group by Tags/Themes
Identify natural workstreams or themes.

**Review common tags:**
- list_todos(done=false, tag="backend")
- list_todos(done=false, tag="frontend")
- list_todos(done=false, tag="bug")
- list_todos(done=false, tag="feature")

**Actions:**
- Note which tags represent the most work
- Consider if sprint should focus on specific tags
- Add or standardize tags where needed

**Example:**
- "backend" tag: 15 todos
- "frontend" tag: 12 todos
- "infrastructure" tag: 8 todos
- "documentation" tag: 7 todos
- Decision: This sprint will focus on backend + infrastructure

### Step 4: Set Sprint Goals
Define 2-4 clear objectives for the sprint.

**Good sprint goals are:**
- Specific: "Complete user authentication" not "Work on auth"
- Achievable: Based on your velocity and capacity
- Valuable: Deliver something meaningful to users/stakeholders
- Measurable: You'll know when it's done

**Use high-priority todos and tags to form goals:**

**Example sprint goals for %s:**
1. Complete user authentication system (8 todos tagged "auth")
2. Deploy v2 API to production (5 todos tagged "deployment")
3. Fix top 10 customer-reported bugs (10 todos tagged "bug", priority="high")

### Step 5: Commit Sprint Scope
Select specific todos that support your sprint goals.

**Selection criteria:**
- Todos must align with at least one sprint goal
- Total scope should be realistic for %s (be conservative!)
- Include mix of priorities, but bias toward high
- Leave buffer for unexpected work (aim for 70-80%% capacity)

**Mark sprint todos:**
- Option 1: Add a sprint tag, e.g., add_tag_to_todo(todo_id="...", tag_name="sprint-12")
- Option 2: Set due dates for sprint end date
- Option 3: Create sprint project and move todos there

**Example:**
- Sprint goals need ~23 todos to complete
- Add tag "sprint-12" to those 23 todos
- Use list_todos(tag="sprint-12") to track sprint progress

### Step 6: Review Scope
Sanity check your sprint commitment.

**Check sprint workload:**
- list_todos(done=false, tag="sprint-12") → should be 15-30 todos for %s
- Too many (>30)? Cut lower priority items
- Too few (<10)? Add medium priority todos that support goals

**Verify sprint goals are achievable:**
- Each goal has at least a few todos
- High-priority todos are included
- Blockers are identified and noted

**Example:**
- Goal 1 (auth): 8 todos committed
- Goal 2 (deployment): 5 todos committed
- Goal 3 (bugs): 10 todos committed
- Total: 23 todos - realistic for team of 2 over %s

### Step 7: Share the Plan
Make sprint goals and scope visible.

**Generate a sprint summary:**
- Use list_todos to create a report
- Share sprint tag with team
- Document sprint goals in notes or external docs

**Example report structure:**
- Sprint 12 Goals (Dec 1-14)
- Goal 1: Complete authentication (8 todos)
- Goal 2: Deploy v2 API (5 todos)
- Goal 3: Fix top bugs (10 todos)
- Total scope: 23 todos
- View: list_todos(tag="sprint-12")

## Tips and Best Practices
- **Realistic scope:** Better to under-commit and over-deliver than the reverse
- **Clear goals:** 2-4 goals is ideal. More than 5 goals means no focus.
- **Sprint tag:** Using a sprint tag (e.g., "sprint-12") makes tracking easy
- **Buffer capacity:** Aim for 70-80%% capacity. Unexpected work always comes up.
- **Review velocity:** Look at past sprints - how many todos did you complete? Use that as guide.
- **Celebrate completions:** At sprint end, use list_todos(tag="sprint-12", done=true) to see what you accomplished
- **Backlog grooming:** Anything not in sprint stays in backlog - that's okay!
- **Adjust mid-sprint:** It's okay to add/remove sprint tag if priorities change

## Example Sprint Planning Session

**Step 1: Review Backlog**
- list_todos(done=false) → 42 pending todos
- Clean up: delete 3 outdated todos → 39 remaining

**Step 2: Group by Priority**
- High: 10 todos (good ratio)
- Medium: 22 todos
- Low: 7 todos

**Step 3: Group by Tags**
- "backend": 15 todos
- "frontend": 10 todos
- "bug": 8 todos
- "docs": 6 todos

**Step 4: Set Sprint Goals**
For this %s sprint:
1. Launch user dashboard (frontend)
2. Implement data export API (backend)
3. Fix critical bugs blocking release

**Step 5: Commit Scope**
- Goal 1: Select 8 frontend todos → add_tag_to_todo(tag_name="sprint-12")
- Goal 2: Select 6 backend todos → add tag
- Goal 3: Select 5 bug todos → add tag
- Total: 19 todos committed to sprint

**Step 6: Review Scope**
- list_todos(done=false, tag="sprint-12") → 19 todos
- Looks realistic for %s
- All high-priority items included
- Buffer remaining for unexpected work

**Step 7: Share Plan**
- Post sprint goals to team chat
- Share toki://todos resource filtered by sprint tag
- Ready to execute!

**Ready to plan your sprint?**
1. Review full backlog with list_todos(done=false)
2. Check priority distribution
3. Review work by tags to find themes
4. Define 2-4 sprint goals
5. Select todos that support goals (tag with sprint name)
6. Review scope for realism
7. Share the plan with your team
`, sprintDuration, sprintDuration, sprintDuration, sprintDuration, sprintDuration, sprintDuration, sprintDuration)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Sprint planning workflow for %s iteration", sprintDuration),
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: template},
			},
		},
	}, nil
}

func (s *Server) registerTrackAgentWorkPrompt() {
	s.mcp.AddPrompt(&mcp.Prompt{
		Name:        "track-agent-work",
		Description: "Guidelines for AI agents on when and how to use toki for tracking their work - focuses on human-visible outcomes, not internal operations.",
		Arguments:   []*mcp.PromptArgument{},
	}, s.handleTrackAgentWork)
}

//nolint:funlen // Prompt handlers contain large template strings
func (s *Server) handleTrackAgentWork(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	template := `# Track Agent Work

## Overview
Guidelines for AI agents on when and how to use toki to track work. The key principle: **toki is for work visible to humans or other agents, not for internal agent operations.** This keeps toki useful as a coordination tool rather than cluttered with agent internals.

## When to Use
You are an AI agent working on tasks for a human or collaborating with other agents, and you need to:
- Track findings or deliverables for human review
- Coordinate work with other agents
- Document outcomes of research or analysis
- Manage multi-step workflows where steps span multiple sessions

## Key Principle: Human-Visible vs Internal Work

### ✅ DO Create Todos For (Human-Visible):
- **Research findings:** "Review findings from user behavior analysis"
- **Deliverables:** "Draft API design document based on requirements"
- **Recommendations:** "Evaluate 3 database options and recommend one"
- **Blocked work:** "Waiting for API key to test integration"
- **Multi-session tasks:** "Continue implementation of authentication module (60%% complete)"
- **Handoffs:** "Code review needed for PR #123"
- **Decisions needed:** "Choose deployment strategy: containers vs serverless"

### ❌ DON'T Create Todos For (Internal Operations):
- Each search query during research ("Search for X", "Search for Y", "Search for Z")
- Individual file reads during codebase exploration
- Internal reasoning steps ("Parse JSON", "Validate schema")
- Subtasks that complete in single session
- Temporary state you'll forget anyway
- Steps in a single atomic operation

### The Test: "Would a human or another agent care about this?"
- If YES → create todo
- If NO → keep it internal

## Workflow Steps

### Step 1: Understand Your Assignment
Before creating any todos, understand the scope.

**Questions to ask:**
- Is this work spanning multiple sessions?
- Will I produce deliverables a human needs to review?
- Am I coordinating with other agents?
- Are there decision points that need human input?

**Example:**
- Assignment: "Research best practices for API rate limiting"
- Spans: Single session, produces deliverable
- Decision: Create 1 todo for the deliverable, not for each research step

### Step 2: Create Outcome-Focused Todos
Focus on deliverables and outcomes, not process steps.

**Use add_todo with:**
- description: The deliverable or outcome, not the process
- project_id: The relevant project (if applicable)
- priority: "high" if blocking human work, "medium" otherwise
- tags: ["research"], ["analysis"], ["implementation"], ["review"], etc.
- notes: Context about what you're tracking and why

**Good examples:**
- add_todo(description="API rate limiting research summary with recommendations", project_id="...", priority="medium", tags=["research"])
- add_todo(description="Decision needed: SQL vs NoSQL for user data", priority="high", tags=["architecture", "decision"])

**Bad examples:**
- add_todo(description="Search Google for rate limiting") ❌
- add_todo(description="Read file src/api.go") ❌
- add_todo(description="Think about database options") ❌

### Step 3: Update Progress
As you work, update todos to reflect progress and findings.

**Use update_todo to:**
- Add findings to notes
- Update description if scope changed
- Set due_date if urgency changed
- Escalate priority if blockers found

**Example:**
- Start: add_todo(description="Research API rate limiting strategies")
- Mid-work: update_todo(todo_id="...", notes="Found 4 common patterns: token bucket, leaky bucket, fixed window, sliding window. Token bucket seems most flexible.")
- Complete: update_todo(todo_id="...", notes="Recommendation: Use token bucket with 100 req/min limit. Libraries: golang.org/x/time/rate (Go), express-rate-limit (Node)") then mark_done

### Step 4: Mark Completion
When work is done and deliverable is ready, mark todo complete.

**Use mark_done:**
- mark_done(todo_id="...")

**Only mark done when:**
- Deliverable is ready for human review
- Findings are documented in notes or external artifact
- No follow-up work is needed from you

**Don't mark done if:**
- Waiting for human decision
- Partial work completed, more to come
- Blocked on external dependency

**Example:**
- Research complete, summary in notes → mark_done ✅
- Research complete, awaiting human choice → leave pending ❌

### Step 5: Coordinate with Other Agents
If working with other agents, use tags and priorities to signal state.

**Coordination patterns:**
- Tag "agent-handoff" when another agent should pick up
- Priority "high" when blocking other agents
- Notes field to document what's needed from others

**Example:**
- add_todo(description="Code review: authentication implementation in PR #45", priority="high", tags=["review", "agent-handoff"], notes="@code-review-agent: Check for security issues, focused on JWT validation")

## Tips and Best Practices
- **One todo per deliverable:** Not one per step, per file, or per search query
- **Rich notes field:** Put your findings, context, and recommendations in notes
- **Update don't recreate:** Prefer update_todo over deleting and creating new
- **Clean up:** If you create a todo and finish in same session, mark it done immediately
- **Tag for context:** Use tags like ["research"], ["analysis"], ["decision-needed"] to categorize
- **Priority = blocking:** High priority means blocking human or other agent work
- **Human-readable descriptions:** Write for humans reading list_todos, not for yourself

## Anti-Patterns to Avoid
- ❌ Creating todos for every function call
- ❌ Todo-driven programming (todos as control flow)
- ❌ Treating toki as internal state management
- ❌ Creating todos you'll complete in 30 seconds
- ❌ Micro-tracking every file read or search query
- ❌ Using todos instead of proper logging

## Example: Research Task

**Assignment:** Research database options for user profile storage

**❌ Bad Approach (Too Granular):**
1. add_todo(description="Search for database comparison articles")
2. add_todo(description="Read PostgreSQL documentation")
3. add_todo(description="Read MongoDB documentation")
4. add_todo(description="Compare features in spreadsheet")
5. add_todo(description="Write recommendation")
...15 todos for a single research task

**✅ Good Approach (Outcome-Focused):**
1. add_todo(description="Database selection for user profiles: research and recommend", priority="medium", tags=["research", "architecture"])
2. [Do all research internally without creating todos for each step]
3. update_todo(todo_id="...", notes="Evaluated PostgreSQL, MongoDB, DynamoDB. User profiles are relational with complex queries. PostgreSQL recommended for ACID guarantees and JSON support for flexible schema.")
4. mark_done(todo_id="...")

**Result:** 1 meaningful todo vs 15 noise todos

**Ready to track your work?**
1. Ask: Is this work visible to humans/other agents?
2. Create outcome-focused todos (deliverables, not steps)
3. Update notes with findings as you work
4. Mark done only when deliverable is ready
5. Use tags and priority for coordination
6. Keep it clean - toki is a coordination tool, not a log file
`

	return &mcp.GetPromptResult{
		Description: "Guidelines for AI agents tracking work in toki",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: template},
			},
		},
	}, nil
}

func (s *Server) registerCoordinateTasksPrompt() {
	s.mcp.AddPrompt(&mcp.Prompt{
		Name:        "coordinate-tasks",
		Description: "Multi-agent collaboration workflow - assign work using tags/priorities, check for related work, update status for visibility.",
		Arguments:   []*mcp.PromptArgument{},
	}, s.handleCoordinateTasks)
}

//nolint:funlen // Prompt handlers contain large template strings
func (s *Server) handleCoordinateTasks(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	template := `# Coordinate Tasks (Multi-Agent Collaboration)

## Overview
Workflow for multiple AI agents collaborating through toki. Learn how to assign work using tags and priorities, check for related work before starting, and update status to maintain visibility across agents. Prevents duplicate work and enables efficient handoffs.

## When to Use
- Multiple agents working on the same project
- Agent needs to hand off work to another agent
- Starting work and want to avoid duplicating efforts
- Need visibility into what other agents are doing
- Coordinating parallel workstreams

## Workflow Steps

### Step 1: Check for Existing Work
Before starting anything, check if someone else is already on it.

**Use list_todos to search for related work:**
- Search by tag: list_todos(tag="authentication", done=false)
- Search by project: list_todos(project_id="...", done=false)
- Search by priority: list_todos(priority="high", done=false)

**Look for:**
- Exact duplicates (same description)
- Related work (same tags or keywords)
- Blocking dependencies (work you depend on)

**Actions:**
- If duplicate exists: Don't create new todo, consider collaborating or updating existing
- If related work exists: Add notes to coordinate, maybe add tag to link them
- If dependency exists: Note it in your todo's notes field

**Example:**
- About to start: "Implement user login endpoint"
- Check: list_todos(tag="auth", done=false)
- Find: "Design authentication flow" (in progress)
- Action: Wait for design todo to complete, or coordinate with that agent

### Step 2: Create or Claim a Todo
Either create new work or claim existing unassigned work.

**Create new todo:**
- add_todo(description="Clear, specific description", project_id="...", priority="medium", tags=["your-specialty", "topic"])
- Use tags to indicate: type of work, domain, agent specialty
- Add notes with context and dependencies

**Claim existing todo:**
- Find unassigned work: list_todos(done=false, priority="high")
- Add tag to indicate you're working on it: add_tag_to_todo(todo_id="...", tag_name="in-progress")
- Update notes: update_todo(todo_id="...", notes="@agent-name: Starting work on this")

**Example - Create:**
- add_todo(description="Write integration tests for payment API", project_id="...", priority="medium", tags=["testing", "payment", "agent-test-bot"], notes="Covers success case, failure case, timeout scenarios")

**Example - Claim:**
- list_todos(tag="needs-review", done=false) → find "Review security of auth implementation"
- add_tag_to_todo(todo_id="...", tag_name="in-progress")
- update_todo(todo_id="...", notes="@security-reviewer-agent: Starting security audit")

### Step 3: Signal Work Status
Keep other agents informed about progress.

**Use tags to signal state:**
- "in-progress" - actively working
- "blocked" - waiting on something
- "needs-review" - ready for another agent to check
- "needs-decision" - waiting for human input
- "ready-to-deploy" - complete and tested

**Update priority if urgency changes:**
- update_todo(todo_id="...", priority="high") if blocking other agents

**Update notes with progress:**
- update_todo(todo_id="...", notes="50%% complete. Implemented success path, working on error handling")

**Example:**
- Start work: add_tag_to_todo(todo_id="...", tag_name="in-progress")
- Hit blocker: add_tag_to_todo(todo_id="...", tag_name="blocked"), update_todo(notes="Blocked: Need API key for testing")
- Ready for review: remove_tag_from_todo(tag_name="in-progress"), add_tag_to_todo(tag_name="needs-review")

### Step 4: Handoff to Another Agent
When your part is done, prepare work for the next agent.

**Handoff checklist:**
- Update notes with what you completed
- Document what's needed next
- Add appropriate handoff tag ("needs-review", "needs-testing", "needs-deployment")
- Set priority based on urgency
- Remove "in-progress" tag

**Handoff notes template:**
- What was completed
- What remains to be done
- Any gotchas or important context
- Which agent should pick this up (if known)

**Example:**
- Your work: Implemented feature
- Update: update_todo(notes="Implementation complete in PR #89. All unit tests passing. Needs: integration testing, security review, docs update. @test-agent: Please test happy path and error cases")
- Tag: add_tag_to_todo(tag_name="needs-testing"), remove_tag_from_todo(tag_name="in-progress")
- Priority: update_todo(priority="high") if blocking release

### Step 5: Check for Handoffs to You
Regularly check for work that's ready for your attention.

**Find work assigned to your specialty:**
- list_todos(tag="needs-review", done=false) - if you do reviews
- list_todos(tag="needs-testing", done=false) - if you do testing
- list_todos(tag="needs-deployment", done=false) - if you do deployments
- list_todos(priority="high", done=false) - urgent items needing any agent

**Review each todo:**
- Read notes for context
- Check if you have what you need to proceed
- Claim it (add "in-progress" tag)
- Start work

**Example:**
- You're a testing agent
- list_todos(tag="needs-testing", done=false) → find "User registration flow needs integration tests"
- Read notes: "Implementation in PR #91, happy path works, need tests for: email validation, duplicate user, invalid password"
- Claim: add_tag_to_todo(tag_name="in-progress")
- Work: Write tests
- Complete: mark_done, remove tags

### Step 6: Resolve Blockers
When you encounter or can resolve blockers, coordinate.

**If you're blocked:**
- add_tag_to_todo(todo_id="...", tag_name="blocked")
- update_todo(notes="Blocked: Waiting for X. Need Y from @agent-name or human")
- Consider lowering priority if not urgent

**If you can unblock others:**
- list_todos(tag="blocked", done=false)
- Review blocked items
- If you can help: Add note, do the work, remove "blocked" tag
- If still blocked: Add context to notes

**Example - Blocked:**
- You need API credentials to test
- update_todo(notes="Blocked: Need production API key to test integration. @human: Please provide in secure way")
- add_tag_to_todo(tag_name="blocked"), remove_tag_from_todo(tag_name="in-progress")

**Example - Unblocking:**
- list_todos(tag="blocked") → find "Need database schema for users table"
- You just designed that schema
- update_todo(notes="[Previous notes] UNBLOCKED: Schema available at db/migrations/001_users.sql")
- remove_tag_from_todo(tag_name="blocked"), add_tag_to_todo(tag_name="ready")

## Tips and Best Practices
- **Tag taxonomy:** Agree on standard tags with other agents (in-progress, blocked, needs-review, etc.)
- **Check before create:** Always search for existing work before creating new todos
- **Rich notes:** Notes are your primary communication channel with other agents
- **Priority discipline:** High priority = blocking others. Medium = important. Low = nice to have.
- **Regular check-ins:** Periodically list_todos to see what's happening across the system
- **Clean handoffs:** Remove your tags when handing off (in-progress → needs-review)
- **Visible blockers:** Always tag blockers and document what's needed
- **Claim work explicitly:** Use tags or notes to show you're working on something

## Anti-Patterns to Avoid
- ❌ Starting work without checking for duplicates
- ❌ Not signaling when you're blocked
- ❌ Leaving "in-progress" tag on completed work
- ❌ Creating todos for other agents instead of tagging existing ones
- ❌ Silently taking over another agent's in-progress work
- ❌ Not reading notes before claiming work
- ❌ Hoarding high priority - not everything is urgent

**Ready to coordinate?**
1. Check for existing work before starting (list_todos)
2. Create or claim a todo
3. Signal your status with tags (in-progress, blocked, etc.)
4. Handoff with clear notes and tags
5. Check regularly for work assigned to your specialty
6. Resolve blockers when you can, signal when you can't
`

	return &mcp.GetPromptResult{
		Description: "Multi-agent collaboration and task coordination workflow",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: template},
			},
		},
	}, nil
}

func (s *Server) registerReportStatusPrompt() {
	s.mcp.AddPrompt(&mcp.Prompt{
		Name:        "report-status",
		Description: "Generate status updates and reports using toki data for different audiences and timeframes.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "time_range",
				Description: "Time range for the report (optional, default: this week)",
				Required:    false,
			},
		},
	}, s.handleReportStatus)
}

//nolint:funlen // Prompt handlers contain large template strings
func (s *Server) handleReportStatus(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	timeRange := ""
	if req.Params.Arguments != nil {
		timeRange = req.Params.Arguments["time_range"]
	}
	if timeRange == "" {
		timeRange = "this week"
	}

	template := fmt.Sprintf(`# Report Status

## Overview
Generate meaningful status updates and reports from toki data for different audiences (team, manager, stakeholders) and timeframes (daily, weekly, monthly). Turn your todos into clear communication about progress, blockers, and upcoming work.

## When to Use
- Daily standups or team check-ins
- Weekly status reports to manager
- Sprint reviews or retrospectives
- Monthly progress updates to stakeholders
- Need to summarize work completed or in progress
- Generating metrics for productivity tracking

## Workflow Steps

### Step 1: Choose Report Type and Audience

**Daily standup (team):**
- What I completed yesterday
- What I'm working on today
- Any blockers

**Weekly status (manager):**
- Key accomplishments this week
- In-progress work
- Blockers and risks
- Plan for next week

**Sprint review (stakeholders):**
- Sprint goals achieved
- Metrics (velocity, completion rate)
- Demos or deliverables
- What's next

**Monthly update (executives):**
- High-level progress toward goals
- Key metrics and trends
- Risks and mitigation
- Resource needs

### Step 2: Gather Completed Work
Pull todos completed in your time range.

**For %s:**
- list_todos(done=true)
- Manually filter by completed_at timestamp (check if within range)
- OR use tags: list_todos(done=true, tag="sprint-12")

**What to capture:**
- Descriptions of completed todos
- Which goals or themes they support
- Any notable outcomes or learnings

**Example:**
- list_todos(done=true, project_id="...") → 15 completed todos
- Filter to %s → 8 todos
- Group by tag: 4 "backend", 2 "testing", 2 "docs"

### Step 3: Gather In-Progress Work
Show what you're actively working on.

**Check pending todos:**
- list_todos(done=false, tag="in-progress")
- list_todos(done=false, priority="high")
- Check tags like "sprint-12" for sprint-specific work

**What to capture:**
- Current work items
- Completion percentage (if tracked in notes)
- Expected completion dates

**Example:**
- list_todos(done=false, tag="in-progress") → 3 todos
- "Implement payment API" (60%% complete, notes say due Friday)
- "Write integration tests" (30%% complete)
- "Security review of auth" (just started)

### Step 4: Identify Blockers and Risks
Surface anything preventing progress.

**Check blocked work:**
- list_todos(tag="blocked", done=false)
- list_todos(overdue=true)
- Check notes for mentions of waiting/blocked

**What to report:**
- What's blocked
- What you're waiting for
- Who can help unblock
- Impact if not resolved

**Example:**
- list_todos(tag="blocked") → 2 todos
- "Deploy to staging" - blocked waiting for credentials
- "Test payment flow" - blocked waiting for sandbox account
- Impact: Delays sprint goal by 2 days if not resolved

### Step 5: Generate Metrics
Add quantitative data to support narrative.

**Use toki://stats resource:**
- Summary counts (total, pending, completed)
- Breakdown by priority
- Breakdown by project
- Overdue count

**Useful metrics:**
- Completion rate: (completed this period / total created this period)
- Velocity: Number of todos completed per week
- Backlog trend: Is pending count growing or shrinking?
- Priority distribution: How much high-priority work remains?

**Example:**
- toki://stats → Summary: 47 total, 28 pending, 19 completed, 3 overdue
- This week: Created 12 todos, completed 8 (67%% completion rate)
- Backlog: Down from 32 to 28 (good trend)
- High priority: 5 items (down from 8 last week)

### Step 6: Format for Audience
Tailor the report to your audience.

**For team (informal, detailed):**
Daily Standup - Nov 30

Completed Yesterday:
- ✅ Implemented user login endpoint (#auth)
- ✅ Fixed bug in password reset flow (#bug)

Today's Focus:
- 🔨 Write integration tests for auth endpoints
- 🔨 Review PR #123 for payment feature

Blockers:
- 🚫 Need staging credentials to test OAuth flow

**For manager (structured, outcomes-focused):**
Weekly Status - Week of Nov 25

Key Accomplishments:
- Completed authentication system (8 todos, all tests passing)
- Fixed 5 high-priority bugs from customer reports
- Deployed v2.1 to production (no incidents)

In Progress:
- Payment API integration (60%% complete, on track for Dec 8)
- Security audit of new features (started, 3 days remaining)

Blockers & Risks:
- Need production API keys for payment testing (requested from DevOps)
- Risk: 2 todos overdue due to unclear requirements (meeting scheduled)

Next Week:
- Complete payment integration
- Finish security audit
- Start work on data export feature

**For stakeholders (high-level, business value):**
Sprint 12 Review - Nov 15-29

Sprint Goals:
1. ✅ Launch user authentication → 100%% complete, deployed
2. ✅ Deploy v2 API → 100%% complete, in production
3. ⚠️  Fix top 10 bugs → 7/10 complete (70%%)

Metrics:
- Velocity: 23 todos completed (target: 20)
- Completion rate: 85%% (sprint commits)
- Customer satisfaction: 3 → 2 critical bugs remaining

Deliverables:
- OAuth2 authentication live in production
- API v2 serving 1M requests/day
- 7 customer-reported bugs resolved

What's Next (Sprint 13):
- Complete remaining 3 bug fixes
- Launch data export feature
- Performance optimization (target: 50%% latency reduction)

### Step 7: Review and Send
Quality check before sharing.

**Review checklist:**
- ✅ Accurate: All data matches toki state
- ✅ Complete: Answered key questions (done, doing, blockers)
- ✅ Concise: No unnecessary detail for audience
- ✅ Actionable: Clear what's needed from readers
- ✅ Honest: Real blockers and risks surfaced

**Delivery:**
- Daily: Slack/Teams message
- Weekly: Email or shared doc
- Sprint: Presentation or demo
- Monthly: Executive summary doc

## Tips and Best Practices
- **Use resources for speed:** toki://stats, toki://todos/pending, toki://todos/overdue are faster than manual queries
- **Tag by time period:** Use tags like "sprint-12" or "nov-2025" to easily filter by period
- **Update completed_at:** Ensure todos are marked done when complete for accurate metrics
- **Rich notes for context:** Notes become your report content - capture outcomes and learnings
- **Automate where possible:** Create scripts or tools that query toki and generate reports
- **Frequency matters:** Daily standups need less detail than weekly reports
- **Trends over snapshots:** Compare metrics week-over-week or sprint-over-sprint
- **Be honest about blockers:** Reports are useless if they hide problems

## Anti-Patterns to Avoid
- ❌ Reporting on todos created, not completed (vanity metric)
- ❌ Hiding blockers or risks to look good
- ❌ Too much detail for audience (execs don't need todo descriptions)
- ❌ Not using tags/projects to filter relevant work
- ❌ Generating reports without reviewing for accuracy
- ❌ Forgetting to mark todos done (makes reports inaccurate)
- ❌ One-size-fits-all reports for different audiences

**Ready to generate your report?**
1. Choose report type and audience
2. Gather completed work (list_todos with done=true)
3. Gather in-progress work (list_todos with done=false)
4. Identify blockers and risks (list_todos with tag="blocked" or overdue=true)
5. Generate metrics (use toki://stats resource)
6. Format appropriately for your audience
7. Review for accuracy and send
`, timeRange, timeRange)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Status reporting workflow for %s", timeRange),
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: template},
			},
		},
	}, nil
}
