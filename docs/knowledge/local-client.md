---
type: knowledge
summary: How the CLI and MCP bridge resolve repository context and project credentials locally.
note_created: 2026-08-02
updated: 2026-08-02
---

# Local client routing

## Resolution order

From the current working directory, `clank` walks upward to the nearest `.clankspace.json`:

```json
{
  "url": "http://127.0.0.1:8091",
  "project": "relaydesk"
}
```

Explicit `CLANKSPACE_URL` and `CLANKSPACE_PROJECT` environment variables override the file. The default URL remains `http://localhost:8080` when neither is present.

## Credentials

`CLANKSPACE_TOKEN` has highest precedence. Otherwise the CLI looks up the exact URL and project pair in the user-local credential file:

```text
${XDG_CONFIG_HOME:-~/.config}/clankspace/credentials.json
```

Store a project token without exposing it in command arguments:

```bash
printf '%s\n' "$CLANKSPACE_TOKEN" | clank auth set --token-stdin
```

The credential directory is mode `0700` and the atomic credential file is mode `0600`. `.clankspace.json` is safe to commit because it contains no secret.

## Agent preflight

`clank context` reports the resolved server, project, pointer path, token source, repository remote, branch, HEAD, and worktree without printing the token. An agent should stop and request one-time human setup when `tokenConfigured` is false.

The stdio MCP bridge uses the same resolution, so harness configuration can be only `clank mcp` when it starts inside a connected repository.
