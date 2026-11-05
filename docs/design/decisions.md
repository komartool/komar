# Decision log — Komar MVP (JVM only)

This page lists every decision taken while designing the Komar MVP, in the order it was taken. Each
row gives the decision in one line and links to the design document that explains it. The working
record of the interview, with every question, option and verbatim answer, is the design note
`.tasks/feature-design/mvp-jvm.md` (git-ignored). The design documents in `docs/design/` are the
source of truth for the design.

Legend for **Status**:

- **settled** — the person chose it.
- **settled, with proposals** — the person chose the structure; some details (a number, a name) are
  my proposals, not confirmed. Those details are listed in
  [scope-and-roadmap.md#open-questions](scope-and-roadmap.md#open-questions).
- **delegated** — the person handed the choice to me ("decide as seems better"). It is my pick, not
  the person's decision.

Linked documents: [spec-language.md](spec-language.md), [output-format.md](output-format.md),
[jvm-runtime.md](jvm-runtime.md), [lifecycle-and-failures.md](lifecycle-and-failures.md),
[scope-and-roadmap.md](scope-and-roadmap.md), [prior-art.md](prior-art.md).

## The guiding principle

In D-17 the person said: "b is good, i like the forgiving design for AI agents". It is not a decision
on any one question, but later options were weighed against it. A forgiving rule omits, skips or
warns where a strict one would fail, and it always tells the agent why.

## Decisions

| ID | Topic | Decision | Status | Explained in |
|---|---|---|---|---|
| D-1 | Attach modes | Launch (`-- <program>`) and attach (`-p <pid>`), both through one in-process Java agent. | settled | [jvm-runtime.md](jvm-runtime.md) |
| D-2 | Probe points | Method entry, normal exit, exception exit, source line with locals, and call site. `where` is structured: "i'd like some consrtuctuion for class, method, line (not a single string with all three)"; "we dont waont to lose entry, exit, exception etc". Follow-up: "i like b" (call sites). | settled | [spec-language.md](spec-language.md) |
| D-3 | Java versions | Target JVMs from Java 8 up. | settled | [jvm-runtime.md](jvm-runtime.md) |
| D-4 | JVMs and OS for `-p` | HotSpot on Linux and macOS. Launch mode works on any JVM that reads `JAVA_TOOL_OPTIONS`. | settled | [jvm-runtime.md](jvm-runtime.md) |
| D-5 | What one probe covers | One probe may select several points of one method; one record per point hit; records of one invocation share a call id. | settled | [spec-language.md](spec-language.md), [output-format.md](output-format.md) |
| D-6 | Class matching | Exact name or wildcard; subtype matching only when the probe opts in (`subtypes: true`). | settled | [spec-language.md](spec-language.md) |
| D-7 | Overloads | Method name plus an optional list of parameter types in Java source syntax. Dotted names match exactly, bare names match the simple name. Generics are ignored, `T...` means `T[]`, type variables match by name. | settled | [spec-language.md](spec-language.md) |
| D-8 | Line and call-site keys | The kind is implied by the keys: `line:` makes a line probe, `call:` a call-site probe, otherwise a method probe. One location kind per probe. | settled | [spec-language.md](spec-language.md) |
| D-9 | Call-site outcomes | Call sites have `before`, `after` (normal return) and `exception` (callee threw). | settled | [spec-language.md](spec-language.md) |
| D-10 | Call target | Matched against the receiver's run-time class, with D-6 rules; static calls against the named class. | settled | [spec-language.md](spec-language.md) |
| D-11 | Constructors | Constructors are probe targets; static initializers are postponed. `this` is not available at constructor entry. | settled | [spec-language.md](spec-language.md) |
| D-12 | Lambdas, anonymous classes | Line and call-site probes on a method also cover lambdas and anonymous or local classes declared in it. | settled | [spec-language.md](spec-language.md) |
| D-13 | Reading values | Fields and exact JDK types are read without running program code. Method calls and `toString()` only when the spec opts in. Follow-up "a": the non-goal becomes "Komar never changes program code; observation runs no application code unless the spec asks". | settled | [spec-language.md](spec-language.md), [jvm-runtime.md](jvm-runtime.md) |
| D-14 | Opt-in for calls | One top-level flag `calls`, default off: "cannot we just make a single flag on top of yaml? if it not set, we forbid calling"; confirmed "good, pick this". | settled | [spec-language.md](spec-language.md) |
| D-15 | Where CEL runs | cel-go checks every expression in the CLI before anything starts; cel-java evaluates in the agent through Komar's own field-reading layer. | settled | [jvm-runtime.md](jvm-runtime.md) |
| D-16 | Shape of `what` | A list; each item is a bare expression (keyed by its text) or `name: expression`. Duplicate keys are a spec error — proposal, not confirmed. | settled, with proposals | [spec-language.md](spec-language.md) |
| D-17 | Missing values in `what` | An item that cannot exist at a point is left out of that record; `null` always means a real null. "b is good, i like the forgiving design for AI agents". | settled | [spec-language.md](spec-language.md) |
| D-18 | Missing values in `when` | The condition is false at that point; the CLI warns before launch. | settled | [spec-language.md](spec-language.md) |
| D-19 | Naming arguments and locals | `args[n]` always; source names under `vars.<name>` when the class keeps them. A missing name follows D-17 / D-18 with a reported reason. | settled | [spec-language.md](spec-language.md) |
| D-20 | Names at call sites | `this`, `args`, `vars` always mean the probed method; the call is `call.target` and `call.args`. Top-level names belong to Komar. | settled | [spec-language.md](spec-language.md) |
| D-21 | Relations between probes | No spans in the MVP. Every record carries `call`, `parent`, `thread` (and `pid`). "ok, for mvp lets do so". Spans, cross-thread links and stack capture are postponed. See "How D-21 evolved" below. | settled | [output-format.md](output-format.md), [scope-and-roadmap.md](scope-and-roadmap.md) |
| D-22 | Value encoding | Plain JSON values; Komar's metadata only in keys starting with `@`. Marker names are working names. | settled, with proposals | [output-format.md](output-format.md) |
| D-23 | Capture limits | Built-in defaults, spec-wide `limits`, per-probe override. Defaults depth 3, 100 items, 1,000 characters, 50 fields ("good defaults."). Per-key merge — proposal, not confirmed. | settled, with proposals | [output-format.md](output-format.md) |
| D-24 | Record kinds | One JSONL stream: `run` header, `hit`, `diag`, `end`. Readers ignore unknown kinds. | settled | [output-format.md](output-format.md) |
| D-25 | Timestamps | Every record has `ts` (ISO-8601 UTC), `t` (ns since agent start) and `seq` (write order). | settled | [output-format.md](output-format.md) |
| D-26 | Hit layout | Fixed envelope plus one `what` object that mirrors the spec. Field names are working names. | settled, with proposals | [output-format.md](output-format.md) |
| D-27 | Several JVMs | Every JVM in a launched tree records; the CLI is the single writer; optional spec-wide `jvm` filter. | settled | [jvm-runtime.md](jvm-runtime.md), [output-format.md](output-format.md) |
| D-28 | Ending a session | Program exit, a signal, or an optional `stop:` condition. In launch mode `stop` also terminates the program (SIGTERM, grace period, SIGKILL). | settled | [lifecycle-and-failures.md](lifecycle-and-failures.md) |
| D-29 | Detach | Restore the original code; a class that cannot be restored has its probes switched off and is reported. | settled | [lifecycle-and-failures.md](lifecycle-and-failures.md) |
| D-30 | `when` fails at run time | The hit does not fire; the first error per probe is a `diag`; the count goes in `end`. | settled | [lifecycle-and-failures.md](lifecycle-and-failures.md) |
| D-31 | Guardrails for calls | Exceptions from calls are contained; probes do not fire inside Komar's own evaluation; after the first call over `call_ms`, calls stop for that probe. Default 50 ms — proposal, not confirmed. | settled, with proposals | [lifecycle-and-failures.md](lifecycle-and-failures.md) |
| D-32 | Exit status | The exit code reports Komar's outcome only; the program's status is in `end`. Codes 0 / 2 / 3 / 4 are working numbers. | settled, with proposals | [lifecycle-and-failures.md](lifecycle-and-failures.md) |
| D-33 | Spec errors | All errors on stderr as `komar: file:line:col: message`, exit 2, output file untouched. | settled | [lifecycle-and-failures.md](lifecycle-and-failures.md) |
| D-34 | Volume limits | Per-probe hit cap and class-match cap in `limits`; Komar never blocks the program. Defaults 1,000 hits and 50 classes — proposal, not confirmed. | settled, with proposals | [lifecycle-and-failures.md](lifecycle-and-failures.md) |
| D-35 | Secrets | No redaction: values are captured as they are. | settled | [output-format.md](output-format.md), [scope-and-roadmap.md](scope-and-roadmap.md) |
| D-36 | Bytecode library | ByteBuddy (with its bundled ASM for line and call-site points). | settled | [jvm-runtime.md](jvm-runtime.md) |
| D-37 | Attach client | The HotSpot attach handshake written in Go, after jattach's design. | settled | [jvm-runtime.md](jvm-runtime.md) |
| D-38 | Containers | Host-to-container attach is postponed; a target in another container is refused with a hint. | settled | [jvm-runtime.md](jvm-runtime.md), [scope-and-roadmap.md](scope-and-roadmap.md) |
| D-39 | Shipping the agent | The agent jar is embedded in the Go binary and extracted to a hash-named cache. | settled | [jvm-runtime.md](jvm-runtime.md) |
| D-40 | Analysis skill | JSON Schema and jq / DuckDB recipes in the MVP; scripts later. | settled | [output-format.md](output-format.md), [scope-and-roadmap.md](scope-and-roadmap.md) |
| D-41 | Verification | "we aim to c surely, but b in scope of mvp implementation. agent checks are postponed. and we surely want TDD approach with many E2E tests with real attaches or sample progams" | settled | [scope-and-roadmap.md#verification-plan](scope-and-roadmap.md#verification-plan) |
| D-42 | Constructor spelling | `method: new`. Accepting `<init>` as an alias — proposal, not confirmed. | settled, with proposals | [spec-language.md](spec-language.md) |
| D-43 | `duration` | A decimal number in milliseconds. | settled | [spec-language.md](spec-language.md) |
| D-44 | Two `hits` keys | The per-probe cap is `limits.max_hits`; `stop.hits` ends the whole run. | settled | [spec-language.md](spec-language.md), [lifecycle-and-failures.md](lifecycle-and-failures.md) |
| D-45 | Non-goals | The list of excluded and postponed items: "Looks right". | settled | [scope-and-roadmap.md](scope-and-roadmap.md) |
| D-46 | Format version marker | Output `run` record always carries `format`; spec `version` is optional. The person: "2. -- decide as seems better". | delegated | [output-format.md](output-format.md), [scope-and-roadmap.md#open-questions](scope-and-roadmap.md#open-questions) |
| Closing | End of the interview | "yes, ready". "Why users need it" is filled with the person's words from `initial-prompt.md`; it was not asked separately. | settled | [README.md](README.md) |

## Rejected alternatives

These options were considered and not chosen. The anchor is the evidence that decided it; "—"
means the person chose otherwise without an external source.

| Option | Why it lost | Anchor |
|---|---|---|
| External debugger over JDI/JDWP | Needs the target started with `-agentlib:jdwp`, so it cannot serve `-p <pid>` on an ordinary JVM (D-1). | https://docs.oracle.com/en/java/javase/25/docs/specs/jpda/conninv.html (unverified); openjdk/jdk `jdk-25+36` `src/hotspot/share/prims/jvmtiManageCapabilities.cpp` |
| Native JVMTI agent with breakpoints and method events | Those capabilities are available only at JVM start, so on attach it gives nothing beyond the Java agent, at the cost of a native build per platform. | openjdk/jdk `jdk-25+36` `src/hotspot/share/prims/jvmtiManageCapabilities.cpp` (`init_onload_capabilities`, `init_onload_solo_capabilities`) |
| JFR method tracing as the probe mechanism | JDK 25+ only; records no arguments or field values. | https://openjdk.org/jeps/520 §Non-Goals |
| `-p <pid>` only | Relies on dynamic agent loading, which JEP 451 plans to disable by default (D-1). | https://openjdk.org/jeps/451 §Description |
| Single-string `where` (`com.shop.Billing#charge:51`) | The person asked for separate class, method and line keys (D-2). | — |
| Reading values through getters and `toString()` by default | Default is field reads; calls only with `calls: true` (D-13, D-14). cel-java's POJO support calls public getters first, so Komar needs its own value layer. | https://docs.datadoghq.com/dynamic_instrumentation/expression-language/ §Overview; github.com/google/cel-java `d9cf3f98` `extensions/src/main/java/dev/cel/extensions/CelNativeTypesExtensions.java:361-381` |
| Evaluating CEL in Go on values shipped from the JVM | Opt-in method calls (D-13) need live objects in the target JVM (D-15). | github.com/google/cel-spec `doc/langdef.md` |
| Komar's own CEL subset in the agent (no cel-java) | The person chose full CEL with cel-java (D-15). | — |
| Typed-node value encoding (`{type, value, fields}`) | The person chose plain JSON with `@` markers (D-22). | dd-trace-java `1c373d58` `dd-java-agent/agent-debugger/src/main/java/com/datadog/debugger/util/MoshiSnapshotHelper.java:45-60` |
| One output file per JVM | Breaks the one-JSONL-file promise and a shared `seq` (D-27). | — |
| ASM only | The person chose ByteBuddy (D-36). | ASM 9.11 on Maven Central |
| JDK Class-File API | Final only in JDK 24; the agent must run on Java 8 (D-3). | https://openjdk.org/jeps/484 (unverified) |
| jattach binary shipped inside Komar, or a `com.sun.tools.attach` helper | An extra native binary per OS and CPU, or a full JDK on the machine, which D-3 rules out (D-37). | github.com/jattach/jattach `src/posix/jattach_hotspot.c:24-82` |
| Name-based secret redaction | The person chose no redaction (D-35). Not built, not rejected for later. | dd-trace-java `1c373d58` `dd-java-agent/agent-debugger/debugger-bootstrap/src/main/java/datadog/trace/bootstrap/debugger/util/Redaction.java:19-170` |
| `inside("request")` function, markers, spans | See "How D-21 evolved". | — |

## How D-21 evolved

D-21 was the longest discussion. It started from a line in `initial-prompt.md`: "somehow to have
conditions relative to other probes, maybe make them linkable".

1. **`inside()`.** The first proposal was a CEL function, `inside("request")`, true while the current
   thread is inside another probe's method. The person liked the idea but asked "what is the
   semantics of inside? its a bit clumsy and not universal". It is a string-keyed special case, it
   means nothing for line probes, and it stops at thread boundaries.
2. **State as data.** The next form was `probes.request.active` and `probes.request.depth`. The person
   found it "clumsy as well" and proposed "markers probes can define and reuse".
3. **Markers.** Three variants were offered: span markers set and cleared by Komar, span markers with
   a captured value, and sticky set/clear flags like Byteman's. The person then asked for a separate
   entity: "maybe we can make some other entity except probes, like span between, idk. lets think".
4. **Research round.** At the person's request ("pls examine more on existing tools, how do they
   define such stuff") four research tracks compared tracing DSLs, APM tools, event-correlation
   languages and Python async. The finding: tool-managed intervals on one thread never leak, while
   cross-thread relations need either context propagation or an explicit key with a time window.
   See [prior-art.md](prior-art.md#relations-between-probes).
5. **Spans.** A separate `spans:` section was designed: same `where` / `when` / `what` keys as a
   probe, read by probes as `spans.<id>.active`. The person rejected the names `around` and `with`,
   then chose probe-like keys.
6. **Back to the need.** The person then said "actually i dont understand if the spans are useful.
   looks like we want them to bind to the stack or maybe have ids, idk. so later we clearly se inner
   events". That split the need in two: seeing nested events later (ids in the output) and filtering
   while recording (spans).
7. **Ids.** The answer to the first need is `call` (one invocation) and `parent` (the nearest enclosing
   probed call on the same thread). The person asked "is this really a working model?" and "what
   would be a better model". Its gaps were laid out: thread hand-offs start new trees, and frames
   already running at attach appear as roots. Four models were compared.
8. **Decision.** The person proposed "maybe m1 + thread id, no?" and confirmed "ok, for mvp lets do
   so". The MVP has `call`, `parent` and `thread` on every record and no spans. Spans for filtering,
   cross-thread links and stack capture are postponed. The span syntax from step 5 is kept as the
   design for a later version.
