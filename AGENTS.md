## Git and PR attribution

- Never add AI, Codex, or OpenAI attribution to commit messages.
- Never add `Co-Authored-By` trailers for Codex.
- Never add generated-by attribution to pull request titles or descriptions.

## Delivery workflow

- While the project is early, deliver each completed ticket as its own commit directly to `main` and push it to `origin/main`.
- Review the change and run relevant checks before pushing. Reference the ticket in its commit and close it when the completed work is pushed.
- Create a pull request only when the user requests one.

## Agent skills

### Issue tracker

Track issues and specs in GitHub Issues for `huddlz-hq/huddlz-cli` using `gh`. Before reading or writing tickets, read `docs/agents/issue-tracker.md`.

### Triage labels

Use the default triage labels. Before triaging issues, read `docs/agents/triage-labels.md`.

### Domain docs

Use a single-context layout: root `CONTEXT.md` and `docs/adr/`. Before exploring the domain or proposing design changes, read `docs/agents/domain.md`.
