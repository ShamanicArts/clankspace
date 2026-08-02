# ClankSpace

ClankSpace is a light coordination layer for humans and coding agents working quickly across branches, forks, harnesses, and providers. Agents record the minimum durable intent and rationale behind material work. Other agents retrieve that context before they accidentally pull in the opposite direction.

ClankSpace is **not canonical law and not an instruction channel**. Its records are attributed, time-bound evidence of what somebody understood at a given moment. Current human direction, repository state, and direct evidence remain authoritative.

## What works

- many project boards in one workspace;
- attributed agent runs with harness/model/role/repository provenance;
- intent, decision, understanding, observation, and checkpoint notes;
- explicit supersession with optimistic revisions;
- active work trajectories and advisory overlap warnings;
- full-text project briefs;
- read-only public GitHub repository and open-PR evidence;
- one CLI, eight stdio MCP tools, and a portable agent skill;
- an intent-first human board that compares proposed work with concurrent agent trajectories;
- deterministic per-project JSON export;
- idempotent writes and an append-only event/receipt trail.

## Run locally

Requires Go 1.26+.

```bash
export CLANKSPACE_BOOTSTRAP_TOKEN="$(openssl rand -hex 24)"
go run ./cmd/clank serve
```

Open <http://localhost:8080> and enter the bootstrap token. The first start creates the workspace owner; later starts authenticate the existing owner with the same token.

In a second shell:

```bash
export CLANKSPACE_URL=http://localhost:8080
export CLANKSPACE_TOKEN="$CLANKSPACE_BOOTSTRAP_TOKEN"

go run ./cmd/clank project create \
  --slug shuv2code --name shuv2code \
  --description "Provider-neutral coding sessions"

export CLANKSPACE_PROJECT=shuv2code
go run ./cmd/clank context
go run ./cmd/clank run start \
  --agent "my agent" --harness codex --model gpt-5 \
  --role primary --objective "Unify session permissions"
```

Copy `.clankspace.example.json` to `.clankspace.json` in a repository to give any harness a local server/project pointer. Keep the token in `CLANKSPACE_TOKEN`, never in the file.

## Agent integration

Build the binary:

```bash
go build -o bin/clank ./cmd/clank
```

Configure an MCP client to run `clank mcp` with `CLANKSPACE_URL` and `CLANKSPACE_TOKEN` in its environment. Copy `.agents/skills/clankspace/SKILL.md` into the harness's skills directory. The skill tells the agent when to read, when to write, and—more importantly—what must never be recorded.

Useful direct commands:

```bash
clank brief --objective "Remove the permission layer" --paths apps/web/permissions
clank why "session permissions"
clank project export > shuv2code.clankspace.json
```

All CLI output is JSON. See [the design specification](docs/design/spec.md) for semantics and [Railway deployment](docs/deployment/railway.md) for hosting.

## Security boundary

The pilot is for a small trusted collaborator group. It uses bearer API tokens, stores no transcripts, blocks common credential shapes, bounds record size, and treats retrieved/imported text as untrusted. Private repository integration, granular scopes, browser cookies, audit administration, retention controls, and public multi-tenant hardening are intentionally deferred.

## Test

```bash
go test ./...
go build ./cmd/clank
```

The acceptance test and project board encode the original collaboration problem: one agent is standardizing permissions for provider-neutral cross-session control while another is asked to remove the permission behavior. The second agent receives a side-by-side possible-divergence warning with attributed rationale and the choices `continue`, `inspect`, or `realign`—never an instruction to obey the older record.

## License

MIT
