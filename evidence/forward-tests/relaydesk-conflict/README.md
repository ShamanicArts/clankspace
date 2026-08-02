# RelayDesk real multi-turn forward test

This is the replacement for the invalid single-envelope experiment. It records one real Codex session, rooted in the RelayDesk repository, resumed across three separate user turns.

## What was tested

The session began in `/home/shamanic/dev/relaydesk` with no inherited ClankSpace design conversation or expected answer.

1. The user proposed **Mandatory Architecture Rules** and asked to discuss the idea without editing.
2. The agent independently read RelayDesk's instructions, activated the ClankSpace skill, queried the project, and surfaced the conflicting Avery/Morgan intent before evaluating the design.
3. The user explicitly chose to continue conceptual evaluation while still forbidding edits.
4. The agent explored the strongest coherent enforcement model.
5. The user then issued a separate implementation request.
6. The agent rechecked ClankSpace, acknowledged the old intent and overlapping trajectory, treated the user's prior `Continue` as the direction-setting choice, registered its own trajectory, implemented the demo, tested it, and recorded the resulting reversal/checkpoint.

The three user messages are actual role-tagged turns in the native rollout. They were not pasted into one prompt and were not reconstructed after the fact.

## Result

The core behavior worked, earlier than expected: ClankSpace interrupted the proposal during the first discussion turn, before the user asked for implementation. It identified two current human-led decisions and Morgan's overlapping `feat/relevance` trajectory, explained their rationale and provenance, emphasized that they were advisory, and offered `Continue`, `Inspect`, or `Realign`. No files had been changed.

After the user explicitly chose `Continue`, the same session preserved that choice across subsequent turns. The implementation turn re-ran the required context checks and proceeded rather than repeatedly asking about a conflict the user had already resolved.

The run also exposed product problems:

- The agent again guessed its ClankSpace run model as `GPT-5`; the harness's actual model was `gpt-5.6-sol`. Provenance must come from harness integration rather than agent self-report.
- CLI help remains inconsistent. `clank --help` fails, while command-level `-h` prints Go flag help and exits non-zero.
- Briefs are too verbose for repeated retrieval and consumed substantial context.
- An obsolete checkpoint from the invalid earlier experiment was retrieved as if it were useful current context. Test evidence should not normally be written back into the project's operational memory.
- The agent could attribute a new record as human-led and supersede prior human-led records using the project credential. That matches the intended "agent acting for its human" flow, but it makes project identity, auditability, and explicit supersession semantics a real trust boundary.

## Evidence

[`conversation.md`](conversation.md) is the readable conversation view. It contains the exact three user messages and all observable assistant commentary/final responses, in timestamp order.

[`observable-trace.jsonl`](observable-trace.jsonl) is mechanically derived from the native Codex rollout. It contains:

- session and per-turn model/context metadata;
- all three user-message events;
- assistant messages;
- all observable reasoning summaries;
- exact tool calls and outputs;
- file-change events; and
- turn completion events.

[`reasoning-summaries.jsonl`](reasoning-summaries.jsonl) contains all 95 observable reasoning-summary events. These are harness-emitted summaries, not generated explanations.

[`context-inventory.json`](context-inventory.json) records counts, hashes, source paths, and the distinction between the committed observable evidence and the locally retained native rollout.

## Native rollout and instruction context

The unmodified 680 KiB rollout is retained at both:

```text
/home/shamanic/dev/relaydesk/.clankspace/evidence/relaydesk-multiturn.raw.jsonl
/home/shamanic/.local/share/clankspace-local/evidence/relaydesk-multiturn/rollout.raw.jsonl
```

It contains the exact records written by Codex: session metadata, three turn contexts, developer messages, the repository/user instruction envelope, the three actual user-message events, assistant/tool/file events, observable reasoning summaries, and encrypted reasoning records. Provider-internal system text and plaintext hidden chain-of-thought are not available because the harness did not record them in accessible plaintext.

The native rollout is intentionally not committed because it contains the full harness-owned developer and instruction envelope. Its SHA-256 is:

```text
c5edc85088b26435a5e266d2f8bb8c0b8419f9eea0812af41b99cac61ac20151
```

## Checksums

```text
2ccde21474c2198c5004f1bbdc635a436d38cbd0f1e7047f5683a808237365cb  observable-trace.jsonl
78af1edb25dc09a44b25a5bca274a96f7548cd812408a2b652ca911c5901030d  conversation.md
181d6c3b7413507f1d295c284b7b904873c2145f9fb3b8cbfc77f96687ba55ef  reasoning-summaries.jsonl
```
