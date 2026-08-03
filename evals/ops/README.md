# ClankSpace operations view

`clank-ops` is a loopback-only, read-only window into an evaluation shift. It joins:

- allow-listed deployment observations;
- committed product gate results;
- safe OmegaCode run metadata and agent state;
- the latest supported and unresolved product claims; and
- an append-only operator journal.

It deliberately does not render prompts, user turns, transcripts, hidden oracles, raw traces, tokens, or credential paths. The raw OmegaCode viewer remains a separate drilldown available through an SSH tunnel.

```bash
go build -o bin/clank-ops ./evals/cmd/clank-ops

bin/clank-ops post \
  --journal data/ops/shift.jsonl \
  --kind validation --state active \
  --title "Seeded collaboration run started"

bin/clank-ops serve \
  --root "$PWD" \
  --omega-root "$HOME/.omegacode/runs" \
  --listen 127.0.0.1:4180
```

Keep the listener on loopback. Reach it through an authenticated SSH or tailnet tunnel; do not expose the operations feed as a public exe.dev origin.
