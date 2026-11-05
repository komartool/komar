# Scope and roadmap — Komar MVP (JVM only)

This page says what the MVP includes, what it leaves out and why, what is still open, how the MVP
will be verified, and what could go wrong. Decision IDs (`D-n`) point to
[decisions.md](decisions.md).

## In the MVP

- **CLI.** `komar run spec.yaml -o out.jsonl -- <program>` launches a program; `komar run spec.yaml
  -o out.jsonl -p <pid>` attaches to a running JVM (D-1). The CLI checks the whole spec before it
  starts anything (D-15, D-33) and is the only writer of the output file (D-27).
- **Runtime.** Target JVMs from Java 8 up (D-3). `-p` works on HotSpot on Linux and macOS; launch
  mode works on any JVM that reads `JAVA_TOOL_OPTIONS` (D-4). The agent is built on ByteBuddy
  (D-36), ships inside the Go binary (D-39), and is attached by Go code (D-37).
- **Probe points.** Method entry, normal exit and exception exit; source lines, including inside
  lambdas and anonymous classes; call sites with `before`, `after` and `exception`; constructors
  (D-2, D-5, D-8, D-9, D-11, D-12).
- **Matching.** Classes by exact name or wildcard, subtypes on request; methods by name with an
  optional parameter list in source syntax; call targets by run-time class (D-6, D-7, D-10).
- **Expressions.** CEL in `when` and `what`, checked by cel-go in the CLI and evaluated by cel-java in
  the agent (D-15). Values are read from fields; program methods run only with `calls: true` (D-13,
  D-14). Missing values are omitted or make a condition false, with a reported reason (D-17, D-18,
  D-19).
- **Output.** One JSONL file per run with `run`, `hit`, `diag` and `end` records (D-24). Hits carry
  `call`, `parent`, `thread`, `pid`, `ts`, `t`, `seq` and a `what` object (D-21, D-25, D-26). Values
  are plain JSON with `@` markers, under default limits (D-22, D-23).
- **Lifecycle and safety.** Optional `stop` conditions (D-28), clean detach (D-29), contained
  failures (D-30, D-31), Komar-only exit codes (D-32), volume caps that never block the program
  (D-34).
- **Analysis skill.** A JSON Schema for every record kind and a `SKILL.md` with jq and DuckDB recipes
  (D-40).

## Non-goals

Confirmed by the person in D-45 ("Looks right").

**Excluded by the rules in `initial-prompt.md`**

- **Changing program code.** The original rule was "instrumentations modifying code" out of scope. In
  D-13 it was narrowed: Komar never changes program code, and observation runs no application code
  unless the spec sets `calls: true`.
- **Analysis commands in the CLI.** Analysis lives in the agent skill (D-40).

**Excluded by evidence, not by choice**

- **GraalVM native executables.** Bytecode cannot be transformed while a native image runs; agents
  only work at image build time (github.com/oracle/graal discussions #6693, a Q&A thread answered by
  GraalVM maintainers).
- **Debugger (JDI/JDWP), native JVMTI and JFR as the probe mechanism.** Each one either cannot attach
  to an ordinary running JVM or cannot capture values (D-1). See
  [decisions.md#rejected-alternatives](decisions.md#rejected-alternatives).

## Postponed, not rejected

| Item | Why it is postponed | Decision |
|---|---|---|
| Python and other runtimes | The person asked for "MVP, JVM only". The spec and output are designed to stay mappable to Python. | request; [prior-art.md](prior-art.md#python-forward-compatibility) |
| Other event kinds (for example the HTTP boundary) | Extensibility rule in `initial-prompt.md`; not needed for the first version. The `kind` field leaves room (D-24). | initial prompt |
| Field read/write probes | Needs rewriting every access site; less value than line probes. | D-2 |
| Throw/catch inside a method | Option not chosen; a line probe in the catch block covers most cases. | D-2 |
| Static initializers | Run once per class, so under `-p` they have nearly always run already. | D-11 |
| Spans: filtering by another probe while recording ("conditions relative to other probes") | The analysis-time need is met by `call` / `parent` ids. The span syntax is kept for later. | D-21 |
| Links when work moves to another thread | Needs instrumenting executors; OpenTelemetry documents misattribution with this approach. | D-21 |
| Opt-in stack capture in `what` | Cost per hit not measured. | D-21 |
| Rate limiting per probe | Hit caps cover the MVP; rate can be added to `limits` later. | D-34 |
| `-p` on Windows and OpenJ9 | Different attach mechanisms; retransform on OpenJ9 late attach not confirmed. | D-4 |
| Attach from the host into a container | The record channel back to `komar` needs design, and it differs between Java 8 and 16+. | D-38 |
| Hiding secrets in captured values | The person chose not to build it. Not recorded as rejected. | D-35 |
| Analysis scripts | Scripts are added once real usage shows which ones agents need. | D-40 |
| Agent-task evaluation (seeded bugs found by an AI agent) | The stated target, postponed beyond the MVP. | D-41 |

## Open questions

Items that need the person, a measurement, or are not knowable now. "Proposal" means my suggestion,
not confirmed by the person.

| # | Question | Kind | Blocks |
|---|---|---|---|
| O-1 | Format version marker: output `run.format` always, spec `version` optional (D-46). Kubernetes requires `apiVersion`; Docker Compose made `version` informative only. | proposal (delegated to me) | spec language and output schema |
| O-2 | Accept `<init>` as an alias for `method: new` (D-42). | proposal | `where` validation |
| O-3 | Default limits `call_ms` 50 ms (D-31), `max_hits` 1,000 and `classes` 50 (D-34). The person confirmed only depth, items, string and fields (D-23). | proposal | defaults; checked by the benchmark |
| O-4 | Per-key merge of probe `limits` over spec-wide `limits` (D-23); duplicate `what` keys are a spec error (D-16). | proposal | spec validation |
| O-5 | Diagnostics also printed to stderr, warnings prefixed `warning:` (D-24, D-33). | proposal | CLI output contract |
| O-6 | Exact `@` marker names (D-22), envelope field names (D-26), exit-code numbers (D-32). | proposal (working names) | output schema, analysis skill |
| O-7 | Small details: regex in class patterns (D-6); class cap per probe or per spec (D-34); SIGTERM-to-SIGKILL grace period (D-28); `jvm` filter keys beyond the main class (D-27); late-loading classes over the class cap (D-34). | open — decided during implementation (D-45) | spec validation and CLI behaviour |
| O-8 | Real cost per hit of advice, CEL evaluation, field walks and opt-in calls; agent jar size after shrinking. | needs experiment | confirming defaults |
| O-9 | Whether DuckDB reads sparse `what` keys as MAP or struct. | needs experiment | analysis recipes |
| O-10 | Java 8 clock precision and thread id reuse. | needs experiment | notes on `ts` and `thread` |
| O-11 | Whether cel-go and cel-java behave identically on the CEL features Komar uses. | needs experiment | correctness of pre-launch checks |
| O-12 | When a future JDK disables dynamic agent loading by default (JEP 451). | unverifiable now | `-p` on future JDKs |

## Unknowns to measure

The MVP benchmark (D-41) is where these get numbers. Until then every default below is a judgment.

- Overhead per hit for each probe kind, with and without a `when` condition.
- Overhead of the field-reading value layer at the default limits (D-23).
- Cost of opt-in method calls and of the `call_ms` guard (D-31).
- Size of the agent jar after relocating cel-java's dependencies and ByteBuddy, and after stripping
  ByteBuddy's unused multi-release variant (D-36, D-39). Estimated around 10 MB, not measured.
- Throughput at which the CLI starts dropping records (D-34).
- Behavior of Java 8: wall-clock precision and thread id reuse (O-10).

## Verification plan

In the person's words (D-41): "we aim to c surely, but b in scope of mvp implementation. agent checks
are postponed. and we surely want TDD approach with many E2E tests with real attaches or sample
progams".

- **Test-first.** Development is test-driven, with many end-to-end tests against real JVMs.
- **Matrix.** Java 8, 11, 17, 21 and the current JDK (D-3), on Linux and macOS with HotSpot (D-4),
  in launch mode and with a real `-p` attach.
- **Golden output.** Small sample Java programs, one per probe kind and edge case: overloads,
  lambdas, constructors, exceptions, recursion, classes compiled without `-g`, opt-in calls, caps.
  Each spec's JSONL is compared with an expected file after masking `ts`, `t`, thread ids and pids.
- **No behavior change.** Each sample's own output and exit code are identical with and without
  Komar. This checks the rule "Komar never changes program code" (D-13).
- **Overhead benchmark.** Reports the cost per probe hit and gives the first numbers for the defaults
  in D-23, D-31 and D-34.
- **CEL conformance.** The cel-spec conformance tests (github.com/google/cel-spec `40a3c90`
  `tests/simple/testdata/*.textproto`) run against both cel-go and cel-java, on the features Komar
  uses (D-15).
- **Postponed: agent tasks.** A handful of seeded bugs that an AI agent must find with Komar, scored
  on success and spec errors. The stated target, not in the MVP.
- **Manual judgment.** A person reads a few real specs and outputs and judges whether they map
  "easily mentally to the spec", as `initial-prompt.md` asks. This cannot be automated.

## Risks

| Risk | Effect | Mitigation in the design |
|---|---|---|
| A future JDK disables dynamic agent loading by default (JEP 451, https://openjdk.org/jeps/451). | `-p` fails unless the target was started with `-XX:+EnableDynamicAgentLoading`. JDK 21+ already prints a warning on the target's stderr. | Launch mode is unaffected. Attach failure exits with code 3 and a clear message (D-32). |
| The HotSpot attach protocol is internal, documented only in source comments. | A JDK change could break the Go attach client (D-37). | The test matrix covers every supported JDK with a real attach (D-41). |
| cel-go and cel-java drift apart. | An expression accepted before launch behaves differently in the agent. | Conformance tests on both implementations (D-15, O-11). |
| Agent jar size. | cel-java pulls guava and protobuf; with ByteBuddy the jar is probably around 10 MB. | Relocation and stripping; size measured in the MVP (O-8). |
| Secrets in the output (D-35). | Passwords and tokens in captured values reach the file and any model that reads it. | Stated plainly in the docs and the analysis skill. Redaction can be added later. |
| Limits of the call model (D-21). | Work moved to another thread starts a new tree with `parent: null`; calls inside frames already running at `-p` attach appear as roots; ids are unique per JVM only. | Documented in the analysis skill; `thread` and `pid` on every record; correlation by captured program values. |
| Missing debug information. | Line probes need `LineNumberTable`; `vars` need `-parameters` or `-g`; ProGuard strips both by default. | A missing name is omitted or makes the condition false, with a reported reason (D-17, D-18, D-19). |
| Opt-in calls change program behavior (D-13). | A getter may lock, load data or throw. | Off by default (D-14); exceptions contained, slow calls disabled (D-31); records marked as called. |
