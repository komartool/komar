# Komar MVP implementation plan

Goal: a working `komar run spec.yaml -o out.jsonl -- <program>` that is good enough to run
experiments and check that the approach is feasible. The design in `docs/design/` stays the source
of truth; this plan only orders the work and cuts the scope.

## Scope decisions

| Area | MVP | Deferred |
|---|---|---|
| Modes | launch (`JAVA_TOOL_OPTIONS`) | `-p` attach, detach/restore |
| Probe kinds | method, constructor, line (incl. lambdas), call-site | field probes, throw/catch (already postponed) |
| CEL in CLI | cel-go parse; variable-availability-per-point check; `calls` flag check; everything typed `dyn` | full type-check, every D-18 warning |
| CEL in agent | cel-java + Komar field-reading value provider | — |
| Values | field reads, exact JDK types, limits, `@` markers | — |
| Opt-in calls | `calls: true` without the `call_ms` guard | slow-call disabling (D-31) |
| Several JVMs | all JVMs in the tree record, `pid` on every record | `jvm` filter |
| Lifecycle | `stop` (after/hits) and program exit, `end` record, drop counting | polished exit codes, stderr diag mirroring |
| Tests | testscript E2E on local JDKs (11, 21, 25; add 8 if installed) | CI matrix, baseline "no behavior change" check on every case |
| Analysis | a short jq/DuckDB README | JSON Schema + skill |

## Repository layout

```
komar/
├── go.mod                 # single Go module, e2e included
├── cmd/komar/main.go
├── internal/
│   ├── spec/              # YAML model + validation
│   ├── celcheck/          # cel-go checks
│   ├── launch/            # process start, JAVA_TOOL_OPTIONS
│   ├── server/            # TCP record channel
│   ├── writer/            # single JSONL writer
│   └── runtime/jvm/       # embedded agent jar (//go:embed), JVM glue
├── runtimes/
│   └── jvm/agent/         # Gradle (Kotlin DSL) + wrapper, Java 8 target, Shadow jar with relocation
├── e2e/
│   ├── e2e_test.go        # testscript entry
│   ├── testdata/*.txtar   # one case per file, flat
│   └── programs/          # shared Java programs
├── bench/                 # overhead benchmark
├── skill/                 # analysis skill (later)
├── docs/
└── Makefile               # build agent jar -> copy into internal/runtime/jvm -> go build
```

A future Python runtime goes to `runtimes/python/` with Go glue in `internal/runtime/python/`.

## Lint and checks

One entry point: `make lint` (all linters) and `make check` (lint + tests).

| Area | Tool | Config |
|---|---|---|
| Go quality | golangci-lint v2 (staticcheck, revive, gocritic, gocyclo, funlen, unused, gofumpt) | `.golangci.yml` |
| Go architecture | go-arch-lint: components = `internal/*` packages, allowed dependencies declared | `.go-arch-lint.yml` |
| Java quality | Error Prone (javac plugin, runs on JDK 21+, `--release 8`) + Checkstyle (naming, size limits) | `runtimes/jvm/agent/build.gradle.kts`, `config/checkstyle.xml` |
| Java architecture | ArchUnit rules as JUnit tests | `runtimes/jvm/agent/src/test/.../ArchitectureTest.java` |
| No comments (AGENTS.md rule 1) | `scripts/no-comments.sh`: fails on `//` or `/* */` in Go and Java sources; allowlist for `//go:` directives and build tags | — |

Wire protocol agent→CLI: loopback TCP, one JSON object per line. The agent formats records; the CLI
adds `seq` and writes them. The agent's sender uses a bounded queue and drops (and counts) when
it is full.

## Phases

Each phase starts with failing E2E cases and ends with them passing.

### Phase 0 — Skeleton
- Go CLI: arguments (`run`, `-o`, `--`), YAML loading, launching the child with `JAVA_TOOL_OPTIONS`
  appended, forwarding exit codes and signals.
- TCP server, single writer, `run` and `end` records.
- Agent: `premain`, reads the spec file, connects, sends `diag hello`.
- Embedded jar extracted to `~/.cache/komar/agent-<sha>.jar`; `KOMAR_AGENT_JAR` overrides it for development.
- testscript harness with custom commands (`javac`, `komar-run`, `jsonl-cmp` with normalization).
- **Exit criterion:** an E2E case produces `run` + `diag` + `end`.

### Phase 1 — Method and constructor probes
- `where` matching: exact or wildcard class, `subtypes`, method name, `params` in source syntax → descriptor, `new`.
- ByteBuddy `Advice` for entry/exit/exception; per-thread call stack → `call`, `parent`, `thread`.
- `what` limited to built-in names (`args`, `this`, `ret`, `exception`, `duration`), no CEL yet.
- Value layer: field walk, JDK types, limits (depth, items, string, fields), `@type`/`@truncated`/`@ref`/`@error`.
- **Exit criterion:** method/ctor/overload/exception cases pass on all local JDKs.

### Phase 2 — CEL (biggest risk; spike first)
- Spike (≤1 day): cel-java evaluating `vars.order.total > 1000` on a POJO with private fields
  through a custom value provider. If this fails, fall back to converting values into CEL maps
  (copy-on-read) and record that decision.
- Agent: compile expressions once per probe, evaluate `when`/`what`; missing variable → key
  omitted or condition false (D-17, D-18); runtime error → false, one `diag` per probe (D-30).
- CLI: cel-go parse, per-point variable availability, `calls` flag, all errors reported at once, exit 2.
- `calls: true`: method calls through reflection, re-entry guard, `called: true` marker.
- **Exit criterion:** value/expression/condition cases pass.

### Phase 3 — Line probes and `vars`
- ASM visitor through ByteBuddy: insert a hook at the `LineNumberTable` offsets; read locals using `LocalVariableTable` ranges.
- `vars.<param>` from `MethodParameters` or LVT; a missing name produces a diag with its reason (`-g`).
- Lambdas through the enclosing method's `invokedynamic` targets; anonymous and local classes through `EnclosingMethod`.
- **Exit criterion:** line cases with `-g`, the default flags and `-g:none`; lambda case.

### Phase 4 — Call-site probes
- Wrap matching invoke instructions: before/after plus a single-instruction exception handler that rethrows unchanged.
- Match by run-time receiver class; static calls match by owner.
- `call.target`, `call.args`, `ret` at after.
- **Exit criterion:** call cases including interface dispatch and a throwing callee.

### Phase 5 — Lifecycle and safety
- `stop: {after, hits}` ends the program (SIGTERM, then SIGKILL after a grace period); `max_hits` and the per-probe cap; `classes` cap.
- Dropped-record counting; `when_errors` in `end`.
- Several JVMs (a program that forks a second `java`).
- **Exit criterion:** lifecycle/limit cases; a check that sample programs' stdout and exit codes match a run without Komar.

### Phase 6 — Feasibility measurements
- Overhead benchmark (JMH or a simple loop): cost per hit by probe kind, with and without `when`.
- Agent jar size after relocation and shrinking.
- A cel-go vs cel-java comparison on the expression subset Komar uses.
- Two or three realistic experiments: a small Spring or plain HTTP app, an agent-written spec, reading the output with jq.
- **Output:** `docs/mvp-findings.md` with numbers and verdicts on open questions O-3, O-8, O-9 and O-11.

## Main risks to watch

1. cel-java custom value provider API (phase 2 spike).
2. Line-probe and call-site bytecode rewriting with correct stack-map frames (let ASM `COMPUTE_FRAMES` / ByteBuddy handle them).
3. Relocating cel-java's protobuf and guava without breaking them (check in phase 0, not later).
4. Agent startup time in build-tool JVMs (Gradle, Maven) loading a ~10 MB jar.
