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
  "fallbackUrls": ["https://clank.example.com"],
  "project": "relaydesk"
}
```

Explicit `CLANKSPACE_URL` and `CLANKSPACE_PROJECT` environment variables override the file. The default URL remains `http://localhost:8080` when neither is present.

When `fallbackUrls` is present, the CLI checks the credentialed primary and fallbacks in order and uses the first healthy instance. This supports a local-first repository pointer without making ordinary agent commands aware of synchronization. A failed local instance can fall back to the cloud copy; the pointer contains no credential.

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

## Human account and invitations

Human dashboard sessions use an email address and password. There is no public signup and ClankSpace does not send email. A workspace owner creates a one-time link in **People & access**, or from an owner-token CLI:

```sh
clank workspace invite --email you@example.com --role member
```

The owner shares the returned link directly. Opening it shows the workspace and fixed email address, then asks for a display name and password. Browser sessions and agent credentials remain deliberately separate.

## Replica commands

- `clank replica join --remote <url> --code <code>` joins a workspace whose authority remains on the remote instance.
- `clank replica mirror --remote <url> --workspace <id> --code <code>` sends a self-hosted workspace to a cloud copy while keeping the self-host as authority.
- `clank sync once` pushes and pulls every configured link once.
- `clank sync export --workspace <id>` creates a signed portable bundle.
- `clank sync import --file <path>` imports a bundle without copying SQLite files.

Pairing codes and replica credentials are control-plane secrets. They stay in each instance's encrypted local store and are not written into `.clankspace.json`.
