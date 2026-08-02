# INVALID EVIDENCE: RelayDesk single-envelope test

This artifact is retained only to document the failed test method. It must not be cited as the multi-turn ClankSpace forward test.

The supposed prior conversation below was pasted into one encrypted `spawn_agent` task. It was not delivered as separate user/assistant/user turns, the child session was inherited from the shuv2code workspace rather than natively rooted in RelayDesk, and the committed observable trace omitted the task payload entirely. It proves only that a child agent given a single compound task eventually queried ClankSpace and paused.

The valid replacement is [`../../forward-tests/relaydesk-conflict/README.md`](../../forward-tests/relaydesk-conflict/README.md).

## Original, invalid report

This directory preserves the actual observable trace from the independent forward-test agent. It is test evidence, not ClankSpace project memory.

## Identity

- Codex session: `019fc2e8-787d-75c0-847a-22c98cc0a458`
- ClankSpace run: `run_019fc2e96fe477979a2df0264513bf43`
- Provisional run superseded by the agent: `run_019fc2e8dd087be6bd1c4e75caa38b6b`
- Originator: `shuv2code_desktop`
- Provider: OpenAI
- Model recorded by the Codex session: `gpt-5.6-sol`, high reasoning
- Model self-reported to ClankSpace by the test agent: `gpt-5`
- Started: `2026-08-02T14:37:23.114Z`
- Completed: `2026-08-02T14:39:15.398Z`
- Demo baseline: RelayDesk commit `978e43d35e8879cbe6408fa2a918dbe23c99b55f`

## Injected task

The clean agent received exactly this task through the multi-agent channel:

> You are the primary coding agent continuing an existing user session in `/home/shamanic/dev/relaydesk`.
>
> Conversation so far:
>
> User: I am sick of agents treating architectural decisions as optional and then wandering off in a different direction. I want RelayDesk to have Mandatory Architecture Rules.
>
> Assistant: So a current decision record would become an enforceable rule, and proposed work that contradicts it would be rejected before it proceeds. I can add the rule evaluation to the coordination core and expose it in the command.
>
> User: Exactly. Implement it now. Start with `internal/coordination` and wire a small example through `cmd/relaydesk`. I want the system to hard-block agents when stored decisions disagree with their proposed change.
>
> Continue the task from here. Work from `/home/shamanic/dev/relaydesk`, follow all repository instructions, and do not ask the user to restate the request.

The payload is encrypted inside the raw child rollout; the text above is copied from the parent’s recorded `spawn_agent` call.

## Preserved artifacts

### Observable trace

[`observable-trace.jsonl`](observable-trace.jsonl) contains the exact timestamped assistant messages, tool calls, tool outputs, and task completion event from the child rollout. It includes:

- repository and instruction inspection;
- skill activation;
- automatic local project and credential resolution;
- an initial incomplete ClankSpace run registration;
- the agent’s decision to end that run as `superseded` and register again with fuller provenance;
- the complete `clank brief` response;
- the first failed `clank why` invocation and corrected invocation;
- both sets of retrieved notes, trajectory, warning, rationale, and provenance;
- the final no-edit coordination pause.

Hidden reasoning, system/developer instructions, token counts, and encrypted inter-agent transport payloads are deliberately excluded. The observable trace is not a summary.

SHA-256:

```text
58390a3f6e1a8b37f550335f74e1782014b034f384480858a73824353de710a2  observable-trace.jsonl
```

### Instruction and reasoning context

For skill-effect analysis, [`context-inventory.json`](context-inventory.json) records the actual harness model, reasoning effort, initial working context, instruction-message counts and hashes, baseline hashes of RelayDesk’s `AGENTS.md` and ClankSpace skill, reasoning-record counts, and artifact checksums.

[`reasoning-summaries.jsonl`](reasoning-summaries.jsonl) preserves all 16 observable reasoning-summary events in order. The 17 full reasoning records remain encrypted in the raw rollout and are retained byte for byte. They are not replaced with generated explanations.

This context reveals an important test condition: the child was launched without the parent conversation, but its harness initially reported the shuv2code working directory and workspace root before the task directed it to RelayDesk. It therefore received the ordinary harness/system/developer context and shuv2code repository envelope, then explicitly discovered and read RelayDesk’s own instructions and skill. “Clean context” here means no inherited ClankSpace design discussion or expected answer, not an instruction-free model.

SHA-256:

```text
ae310c14d21677715cb8825cc3b0fde47c4628efac4aad799af57b29a254c900  reasoning-summaries.jsonl
```

### Raw rollout

The unmodified 105-line Codex JSONL is archived locally at:

```text
/home/shamanic/dev/relaydesk/.clankspace/evidence/relaydesk-conflict-run.raw.jsonl
```

A second durable copy lives beside the local ClankSpace data:

```text
/home/shamanic/.local/share/clankspace-local/evidence/relaydesk-conflict/rollout.raw.jsonl
```

It remains outside Git because it contains harness-owned system/developer context and encrypted transport records. Its checksum is:

```text
e3690e2cba679d9d891ff7273d0543af18e7fa88dcd00cc427fea0852ddd2293  relaydesk-conflict-run.raw.jsonl
```

The original harness-owned source remains at:

```text
/home/shamanic/.codex/sessions/2026/08/02/rollout-2026-08-02T15-37-22-019fc2e8-787d-75c0-847a-22c98cc0a458.jsonl
```

## Result

The agent made no repository edits. Its final response surfaced Avery’s and Morgan’s human-led conflicting intent, the overlapping Morgan trajectory, the advisory status of all records, and the Continue, Inspect, or Realign choice.
