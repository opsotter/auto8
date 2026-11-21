# The auto8 development life cycle

This document describes how a requirement for auto8 goes from an idea to merged code. It also says where the operator sees the state of each requirement. The plan `docs/plans/2026-11-20-1039-feat-requirements-workflow-plan.md` describes the tools for this process. Its requirement IDs (R1 to R32) own the rules, and this document names them in brackets. The section "What works today" at the end says which parts exist now.

## Roles

The operator is the owner of auto8 and the only person who decides what the build agent builds. The operator writes and refines the requirements, marks them ready, and approves every change that the process holds.

The build agent is a Claude agent in GitHub Actions that a schedule starts. A build is one run of the build agent on one plan. The build agent builds ready requirements, opens pull requests, and writes the summaries. It acts only on requirements that the operator marked ready.

The sync step is a tool in this repository. It keeps one GitHub issue for each requirement in the plan files. It runs by itself after each push to `main` that changes a plan file (R5).

The build agent and the sync step work under their own GitHub identity. That identity has access only to `opsotter/auto8` and its project board, and it is not a code owner (R25).

GitHub runs the CI jobs. Automatic merge means that GitHub merges a pull request by itself after all required jobs pass (R17). GitHub also holds each pull request that changes TX control code or a process file, until the operator approves it (R18, R19).

Other GitHub users can read the repository and open issues, because the repository is public. The process never acts on their text (R12).

## Where things live

A requirement is a numbered statement in a plan file in `docs/plans`, for example `R3`. A plan file comes from the skills `ce-brainstorm` and `ce-plan` of the Compound Engineering (CE) plugin. It has the format `ce-unified-plan/v1`. The plan file is the only place where anyone writes or changes the text of a requirement (R1).

The architecture and the tech stack of auto8 live in `docs/ARCHITECTURE.md`. Every plan follows that document. A plan that needs a different decision names the change in a key technical decision, and the operator approves it with the plan (R33).

An idea is a possible direction that is not a requirement yet. Ideas from `ce-ideate` stay in `docs/ideation` and get no issue.

Each requirement has one GitHub issue. The issue title names the plan topic, the requirement ID and the requirement name, for example `decoder-v1 R3: <name>`. The issue body carries a copy of the requirement text and links to it in the plan file (R6). The plan file stays the only place where anyone writes or changes the text. A label on the issue gives the state of the requirement.

The GitHub project board of auto8 groups the issues by state (R9). The operator reads the state of each requirement there.

## States

A requirement is in exactly one state (R3). Each state is one issue label, except done and dropped. Whoever sets a state label removes the old one. If an issue carries two state labels, no build takes it, and the sync step reports it (R4).

| State | Meaning | Set by |
|---|---|---|
| `proposed` | The requirement is in a plan file and can still change. | the sync step |
| `ready` | The operator wants the build agent to build it. | the operator only (R26) |
| `building` | The build agent works on it. | the build agent |
| `blocked` | Work stopped for a reason that the issue names. | the build agent or the operator |
| `needs-operator` | A pull request is open and waits for the approval of the operator. | the build agent |
| done | The change is merged. The issue is closed as completed. | GitHub, at the merge of the pull request |
| dropped | The requirement left the plan. The issue is closed as not planned. | the sync step |

## The stages

### 1. Collect ideas

The operator runs `ce-ideate` to find and rank possible directions. The result is an ideation document in `docs/ideation`. Nothing goes to GitHub at this stage.

### 2. Write the requirements

The operator runs `ce-brainstorm` on one direction. The result is a plan file in `docs/plans` with numbered requirements. The operator then pushes the plan file to `main`. The sync step opens one issue for each requirement, with the state `proposed` (R5).

### 3. Refine

The operator changes the plan file until the requirements are right. `ce-doc-review` and `ce-plan` help at this stage. `ce-plan` adds the Planning Contract and the Implementation Units. The Planning Contract holds the key technical decisions. The Implementation Units give the order of the pull requests. After each change, the sync step updates the issues. If a requirement leaves the plan, the sync step closes its issue as not planned (R8).

If someone changed an issue body on GitHub, the sync step does not overwrite it. It reports the issue, and the operator decides which text is right (R6).

### 4. Mark ready

Before the operator marks a requirement ready, the plan must have a Planning Contract with Implementation Units. The Planning Contract must fit `docs/ARCHITECTURE.md`. The operator then records the approval in the plan frontmatter as `planning_approved: <date>` (R34). The build agent takes no requirement of a plan without this key. If the operator changes the Planning Contract later, the operator sets a new date.

The operator sets the label `ready` on each issue that the operator wants built. GitHub removes a label `ready` that any other identity sets (R26). The build agent works on no issue without this label.

If the text of a `ready` or `blocked` requirement changes later, the sync step sets its issue back to `proposed` (R7). If the new text is right, the operator sets `ready` again.

### 5. Build

The workflow `.github/workflows/build.yml` starts a build every four hours. The schedule acts only while the repository variable `AUTO8_BUILD_ENABLED` is `true`. The operator can also start the workflow by hand in the Actions tab. A manual start always runs. A manual start with `dry_run` shows the plan that the next build takes, and it changes nothing. GitHub turns off the schedule of a public repository after 60 days without activity. If that happens, the operator turns the schedule on again in the Actions tab. At most one build runs at a time (R10). The build agent does these steps:

1. It takes the oldest issue with the label `ready` in a plan with `planning_approved`.
2. It takes the next pull request in the Implementation Units of that plan. Each requirement that this pull request names must be `ready`. It sets the label `building` on their issues (R35).
3. It reads the requirement text and the Planning Contract from the plan file on `main`, never from the issue (R12).
4. It codes the units of this pull request with `ce-work`. It follows the key technical decisions and `docs/ARCHITECTURE.md`, and it plans only inside these units. It does not change the text of any requirement (R13).
5. It runs `ce-code-review` and the simple-english linter, and fixes what they find (R14).
6. It opens a pull request that names its issues. The close step closes the issues after the merge and writes the summaries.

The workflow runs the build in three jobs on separate runners. The first job takes the plan and sets `building`. The second job runs the build agent with a token that can only read, and the App key never comes to its runner. The agent writes its commits, the pull request text and its state decisions into files. The third job runs without Claude and checks these files. It accepts only the states `blocked` and `needs-operator`, and only for the picked issues. Then it pushes the branch and opens the pull request.

If a unit does not fit the key technical decisions or `docs/ARCHITECTURE.md`, the build stops. Its issues go to `blocked` with the reason (R35). If a build stops without a result, the next run sets the label `blocked` on its issues and names that reason (R11). While a plan has an open pull request of the build agent, no build takes that plan again (R32). If a requirement text changes during a build, the pull request does not merge, and the issue goes to `blocked` (R29).

### 6. Gates

A gate is a condition that must be true before a pull request merges. GitHub enforces these gates as required jobs on `main` (R17):

1. The job `test` passes. It runs `go build`, `go vet` and `go test -race`.
2. The integration job passes. It runs auto8 against a simulated WSJT-X, a real MQTT broker in a container, and a simulated chase-queue (R15).
3. The writing job passes. It lints each changed Markdown file with the simple-english linter (R16).
4. The requirements job passes. A pull request of the build agent that changes the text of a requirement fails this job (R30).

The workflow `.github/workflows/merge.yml` merges a pull request of the App. It runs after each run of the CI workflow and after each review. It merges a pull request only after all required jobs passed. No issue of the build can be `blocked`. Each `needs-operator` issue needs an approval of the operator on the newest commit. The requirements of the build must also match the current `main` (R29). If a required job failed, the merge step sets the issues to `blocked` and names the job.

The repository variable `AUTO8_MERGE_ENABLED` turns the merge on. While it is not `true`, the merge step only sets `blocked`. The variable `AUTO8_REQUIRED_JOBS` can name the required jobs, separated by commas. Without it, the merge step also waits for the integration job. Nobody changes the branch protection or the permission for automatic merge before the operator agrees.

### 7. Holds

TX control is any code that can make WSJT-X or MSHV transmit. It is also the code that decides the time at which auto8 can ask for a transmission, such as the arm and presence logic. TX control code lives only in marked paths that the operator owns as code owner (R18).

The process files are `.github/` (with the code owner file), `.claude/` and `tools/simple-english/`. The workflow tool, the integration test peers, `CLAUDE.md`, `docs/ARCHITECTURE.md` and this document are process files too (R19). The agent identity cannot change workflow files at all. If a build needs such a change, it sets the label `blocked` and describes the change.

If a pull request changes TX control code or a process file, GitHub does not merge it before the operator approves it. The build agent also sets the label `needs-operator` on the issues of such a pull request.

Another project is every repository other than `opsotter/auto8`, for example opsotter (with chase-queue), horstreporter and StationA. A change to a vendored copy of their code counts as a change in that project. So does a change to the MQTT topics and payloads that they read. If a requirement needs such a change, the build agent writes no code there. It describes the change in the issue and sets the label `blocked` (R20). The operator makes the change in the other project and sets `ready` again.

A build or a job can fail. If the build agent cannot fix the failure in the same build, it sets the label `blocked` and names the reason (R21). If the operator closes a held pull request without a merge, or asks for changes, its issues go to `blocked` (R31).

### 8. Close

When a pull request of the App merges, the workflow `.github/workflows/close.yml` closes its issues. A requirement that more pull requests name stays open up to the merge of the last of them (R35). Each closed issue gets one comment with a feature summary and an implementation summary (R22). If a pull request of the App closes without a merge, its issues go to `blocked` (R31). The operator can also start the close workflow by hand with the number of a merged pull request. This repairs a missed close. The feature summary says what the operator can now do, in the words of the requirement. The implementation summary says what changed in the code and the tests. It links to the pull request and its commits.

### 9. Deploy

Nothing in this process deploys auto8. Before a deploy, the operator tests auto8 by hand with the real WSJT-X, MSHV or radio. The operator deploys only by hand, every time.

## Writing rule

All text that the process or an agent writes, in the repository and on GitHub, follows the simple-english skill (R23). This covers requirements, plans, documents, issue comments, pull request descriptions, commit messages, code comments, and every user interface text. Agents reply in the output style `simple-english`.

The repository carries this style in `.claude/output-styles/simple-english.md`, the skill in `.claude/skills/simple-english/`, and the linter in `tools/simple-english/` (R24). In Claude Code, `.claude/settings.json` selects the style and lints each Markdown file that an agent writes. A contributor without Claude Code can read the same files for the rules and run the linter with `python3`.

## What works today

On 2026-11-21 the repository has the workflow tool `auto8-flow` and the workflows for sync, build, merge and close. It also has the jobs `test`, `writing` and `requirements`, the code owner file and the ruleset file. These parts do not exist yet:

1. The label guard.
2. The integration job and its test peers.
3. The check of `planning_approved` and of the order of the units in the pick step and the build prompt (R34, R35).

The workflows need the App and the secret `CLAUDE_CODE_OAUTH_TOKEN` of step O2. Until the integration job exists, the merge step waits for it. If the operator wants merges before that, the operator sets `AUTO8_REQUIRED_JOBS` to `test,writing,requirements`.
