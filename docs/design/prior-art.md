# Prior art — research behind the Komar MVP

This page summarizes the research done while designing the MVP: which tools solve similar problems,
what Komar borrows from them, and the sources that decisions rely on. Decision IDs (`D-n`) point to
[decisions.md](decisions.md).

Labels:

- **checked** — read in a primary source during the design.
- **unverified** — reported by a research subagent and not re-read. Treat as a lead.

All GitHub star counts were collected by research subagents on 2026-10-09 and are **unverified**.

## Tools and libraries

| Tool / library | Popularity (unverified) | What it does | Komar's relation |
|---|---|---|---|
| Arthas | 37,576 GitHub stars | `watch` / `trace` / `line` commands on a running JVM; OGNL conditions and captures (`params`, `target`, `returnObj`, `throwExp`); watch points before / normal exit / exception (`-b` / `-s` / `-e`); subclass matching on by default; match limit 50 classes, hit limit 100 by default. | Design borrowed: separate entry / exit / exception points (D-5, D-9), line probes with locals (D-2), match and hit caps (D-34). Rejected: free method calls in expressions. |
| BTrace | 5,994 GitHub stars | `@OnMethod` with separate `clazz`, `method`, `type` and `@Location(Kind...)`; 17 location kinds including `CALL`; `+Type` for subtypes; `@TLS` per-thread state; a verifier that bans side effects. | Design borrowed: structured location (D-2), opt-in subtypes (D-6), call-site kind (D-2), per-thread pairing (D-21). |
| Byteman | 515 GitHub stars, LGPL-2.1 | ECA rules with `AT ENTRY / EXIT / LINE / INVOKE / THROW`; flags, counters, LinkMaps; `callerEquals`; `setTriggering(false)` against recursive triggering. | Reference only. Borrowed: no probes fire inside Komar's own evaluation (D-31). |
| Datadog Dynamic Instrumentation (dd-trace-java) | 736 GitHub stars | Structured `Where` (`typeName`, `methodName`, `signature`, `lines`) shared by Java and Python; expression language with no method calls; capture limits depth 3 / 100 items / 255 chars / 20 fields; `@duration` in ms; name-based redaction; span and trigger probes. | Design borrowed: structured `where`, optional signature (D-7), limits (D-23), `duration` in ms (D-43). Rejected: typed-node encoding (D-22), redaction (D-35). Not usable directly: needs Datadog's backend. |
| Lightrun | proprietary | Dynamic logs and snapshots; "Dynamic Traces" group snapshots on one thread. | Reference only. |
| Google Cloud Debugger (Java) | 92 GitHub stars, archived | Java-syntax conditions; method calls run in a sandboxed bytecode interpreter with quotas. | Reference only. |
| OpenTelemetry Java agent | 2,642 GitHub stars | Isolated agent class loader with shaded dependencies; carries context across executors. | Design borrowed: class-loader isolation (unverified). Reference for cross-thread propagation and its documented misattribution (D-21). |
| JFR method tracing (JEP 520) | JDK feature | Method timing and tracing by filter, JDK 25+; no arguments or field values. | Rejected as the probe mechanism (D-1); possible future event source. |
| JFR `@Contextual` | JDK 25 feature | A field that applies to all events in the same thread for the duration of an event. | Reference for scoped context (D-21). |
| jattach | 1,154 GitHub stars, Apache-2.0 (checked) | Native attach client for HotSpot and OpenJ9, Linux / macOS / FreeBSD / Windows, containers, no JDK needed. | Design borrowed: the Go attach client follows its HotSpot handshake (D-37). Not shipped as a binary. |
| ByteBuddy 1.18.x | — | Bytecode advice and agent setup; Apache-2.0; runs on Java 8+, supports new class-file versions automatically since 1.17. | Dependency (D-36). |
| ASM 9.11 | — | Low-level bytecode library, about 0.26 MB. | Used through ByteBuddy; ASM-only option rejected (D-36). |
| cel-java (`dev.cel:cel` 0.14.0) | 294 GitHub stars | CEL in the JVM, Java 8+; pulls guava and protobuf-java; its POJO extension calls public getters first. | Dependency in the agent, with Komar's own field-reading layer (D-15). |
| cel-go | 3,121 GitHub stars | CEL in Go; checked-AST serialization. | Dependency in the CLI for pre-launch checks (D-15). |
| cel-python | 173 GitHub stars | Pure-Python CEL; conformance work not complete (unverified). | Reference for a future Python runtime. |
| Grafana Alloy | — | Embeds async-profiler in a Go binary and extracts it to a hash-named temp folder. | Design borrowed for shipping the agent jar (D-39). |
| Sigma correlation rules | — | A separate YAML document that relates event rules by `group-by` key within a mandatory `timespan`. | Reference for postponed cross-thread relations (D-21). |
| Elastic EQL | — | `sequence by <key> with maxspan=... [..] [..] until [..]`; missing-event clauses need `maxspan`. | Reference (D-21). |
| DTrace | — | Thread-local `self->x`; speculative tracing (`speculate` / `commit` / `discard`). | Reference for per-thread state and for "record now, keep or drop later" (D-21). |
| asciicast v2 | — | JSONL with a header on line 1 and events after it; readers ignore unknown event codes. | Design borrowed for record kinds (D-24). |

## Relations between probes

The second research round (D-21) compared how tools relate one instrumentation point to another.

| Kind | Who has it | Lifecycle | When the end never comes |
|---|---|---|---|
| Interval around a method | SystemTap `@entry`, BTrace `@Duration`, Arthas `trace`, Datadog `SpanProbe`, Glowroot trace entries, Elastic `trace_methods`, JFR `begin` / `end` | The tool opens it at entry and closes it at return or throw. | Cannot happen: the tool closes it. |
| Interval as context for other events | JFR `@Contextual` (checked) | Same thread, from the start to the end of the event. | Ends with the event. |
| Separate interval entity | Lightrun Dynamic Trace (unverified; the overview page returned 404) | Tool-managed, limited to one thread. | Max hits or a timeout. |
| Sticky set / clear state | DTrace `self->x`, BTrace `@TLS`, Byteman flags, OpenTelemetry Context attach / detach | The user clears it. | Leaks; every tool warns about it. |
| State keyed by a value | DTrace and bpftrace maps, Byteman LinkMaps, Sigma `group-by`, EQL `sequence by`, Flink `keyBy` | Explicit key, usually with a time window. | Dropped at the end of the window, or evicted. |
| Context that follows work across threads | OpenTelemetry Java agent (checked) | Captured at `submit`, restored in `run()`. | Documented misattribution: it "can inadvertently propagate the execution context of one request to a thread handling an entirely unrelated request". |
| Record now, decide later | DTrace speculation (checked), Arthas `trace` (unverified) | Explicit commit or discard. | Buffers are finite; failures are counted. |

Conclusions that shaped D-21:

- Every tool that relates points within one call scopes it to one thread and closes the interval
  itself. That is the only form that never leaks.
- Following work across threads needs executor instrumentation, and OpenTelemetry documents
  misattribution in practice. No tracing DSL does it declaratively.
- Relating a start and an end by a key always comes with an explicit key and a mandatory time
  window.

The MVP takes none of these as a spec feature. It puts `call`, `parent` and `thread` on every record
so that nesting can be rebuilt during analysis (D-21).

## Python forward compatibility

Python is not in the MVP, but the spec and the output must map onto it later.

- **Probe points.** `sys.monitoring` (CPython 3.12+) has an event for every MVP point: `PY_START`
  (entry), `PY_RETURN` (exit, with the return value), `PY_UNWIND` (exception exit), `LINE`, `CALL`
  (before a call). "After a call" has a direct event only for non-Python callables (`C_RETURN` /
  `C_RAISE`). Tool ids are limited to 0–5. (checked:
  https://docs.python.org/3/library/sys.monitoring.html)
- **Attach.** `sys.remote_exec` (3.14+) runs a script in a running interpreter and needs
  ptrace-level privileges. (unverified)
- **Locations.** Python's natural key is module plus qualified name; Datadog's Python tracer reads
  the same `Where` as Java, treats `typeName` as the module and ignores `signature`. (unverified)
- **Threads and tasks.** asyncio tasks copy the context when they are created and keep it across
  `await`. `ThreadPoolExecutor.submit` does not carry context. (unverified)
- **Thread context inheritance.** In 3.14 a new thread copies the caller's context only with
  `-X thread_inherit_context=1`; the default is 1 on free-threaded builds and 0 otherwise. (checked:
  https://docs.python.org/3/using/cmdline.html)
- **Generators.** PEP 568 (generator-local context) has status Deferred, so a context value set inside
  a generator leaks to its consumer between yields. (checked: https://peps.python.org/pep-0568/)
- **Implications for the design.** Program names live under `vars.` (D-19), method signatures are
  optional (D-7), `method: new` can map to `__init__` (D-42), and "same thread" in the call model
  would become "same task" on Python (D-21).

## Sources

Condensed from the design note. URLs and repository anchors are kept verbatim.

### JVM mechanism and attach

| Source | Anchor | What it established | Label |
|---|---|---|---|
| `java.lang.instrument`, JDK 21 | https://docs.oracle.com/en/java/javase/21/docs/api/java.instrument/java/lang/instrument/package-summary.html §Starting an agent | Agents start at launch (`-javaagent`) or in a running JVM (`Agent-Class`); `Can-Retransform-Classes`. | checked |
| JEP 451 | https://openjdk.org/jeps/451 §Description | JDK 21+ warns on dynamic agent loading; a future release disables it by default unless `-XX:+EnableDynamicAgentLoading`. | checked |
| HotSpot JVMTI capabilities | openjdk/jdk tag `jdk-25+36` `src/hotspot/share/prims/jvmtiManageCapabilities.cpp` | Breakpoint, method entry/exit and local-variable capabilities are start-up only; retransform is always available. | checked |
| JEP 520 | https://openjdk.org/jeps/520 §Non-Goals | JFR method tracing records no arguments or field values. | checked |
| JVMTI 21 spec | https://docs.oracle.com/en/java/javase/21/docs/specs/jvmti.html §JAVA_TOOL_OPTIONS | The VM reads `JAVA_TOOL_OPTIONS` to start agents when the command line is not accessible. | checked |
| HotSpot attach listener | openjdk/jdk tag `jdk-21-ga` `src/hotspot/share/services/attachListener.hpp` | Attach arguments are capped at 1,024 characters, so the spec is passed as a file path. | checked |
| jattach | github.com/jattach/jattach master `LICENSE`; `src/posix/jattach_hotspot.c:24-82` | Apache-2.0; `.attach_pid` file, `SIGQUIT`, `.java_pid<pid>` socket; `load instrument`. | checked |
| `Instrumentation.retransformClasses` / `removeTransformer`, JDK 21 | https://docs.oracle.com/en/java/javase/21/docs/api/java.instrument/java/lang/instrument/Instrumentation.html | Retransformation starts from the initial class bytes; active frames keep the old code; a removed transformer may still be called. | checked |
| JEP 444 | https://openjdk.org/jeps/444 §Thread-local variables | Virtual threads always support `ThreadLocal`. | checked |
| `java.lang.Thread`, JDK 21 | https://docs.oracle.com/en/java/javase/21/docs/api/java.base/java/lang/Thread.html | Thread ids are unique and never change; `Thread.stop` is deprecated for removal. | checked |
| OpenJ9 Attach API | https://eclipse.dev/openj9/docs/attachapi/ §Securing the Attach API | OpenJ9 uses its own attach mechanism. | checked |
| GraalVM and Java agents | github.com/oracle/graal discussions #6693 | No bytecode transformation at native-image run time. | checked (Q&A thread) |
| JDWP process attach | https://docs.oracle.com/en/java/javase/25/docs/specs/jpda/conninv.html | JDI attach needs `-agentlib:jdwp` at start. | unverified |
| Class-File API | https://openjdk.org/jeps/484 | Final in JDK 24. | unverified |
| ByteBuddy | github.com/raphw/byte-buddy master `README.md`; https://javadoc.io/static/net.bytebuddy/byte-buddy/1.18.14/net/bytebuddy/asm/Advice.html | Java 8+ runtime, new Java versions supported automatically; exit advice runs on exceptions; `this` unavailable at constructor entry. | checked |
| Jar sizes | Maven Central, fetched 2026-10-09 | `byte-buddy-1.18.14.jar` 4,658,001 B; ASM 9.11 about 0.26 MB; guava 33.6.0-jre 3,064,793 B; protobuf-java 4.35.1 1,843,510 B. | checked |
| Go `embed` | https://pkg.go.dev/embed | A file can be embedded in a Go binary at build time. | checked |
| Grafana Alloy | github.com/grafana/alloy main `internal/component/pyroscope/java/asprof/asprof.go` | Embeds and extracts a JVM-side artifact to a hash-named temp folder. | checked |
| OpenTelemetry Java cross-thread propagation | github.com/open-telemetry/opentelemetry-java-instrumentation main `docs/misc/inter-thread-context-propagation.md` | Context carried through executors, with documented misattribution. | checked |

### Class files and the Java language

| Source | Anchor | What it established | Label |
|---|---|---|---|
| JVM Specification SE 21, ch. 4 | https://docs.oracle.com/javase/specs/jvms/se21/html/jvms-4.html §4.3.3, §4.6, §4.7.3, §4.7.9.1, §4.7.12, §4.7.13 | Method descriptors; `ACC_VARARGS`, `ACC_BRIDGE`; exception table ranges; generic signatures in the optional `Signature` attribute; `LineNumberTable` and `LocalVariableTable` are optional. | checked |
| JVM Specification SE 21, ch. 6 | https://docs.oracle.com/javase/specs/jvms/se21/html/jvms-6.html §6.5 invokevirtual, invokeinterface | The method run is selected from the receiver's run-time class. | checked |
| JLS SE 21 §8.1.2, §13.1 | https://docs.oracle.com/javase/specs/jls/se21/html/jls-8.html §8.1.2; https://docs.oracle.com/javase/specs/jls/se21/html/jls-13.html §13.1 | Generic parameterizations share one class; a call instruction names the erased compile-time type. | checked |
| javac, JDK 21 | https://docs.oracle.com/en/java/javase/21/docs/specs/man/javac.html §Standard Options | Line numbers by default; local names need `-g`; parameter names need `-parameters`. | checked |
| ProGuard attributes | https://www.guardsquare.com/manual/configuration/attributes §Optional attributes | Obfuscation strips `Signature`, `LineNumberTable`, `LocalVariableTable` and `EnclosingMethod` by default. | checked |
| Lambda translation | https://cr.openjdk.org/~briangoetz/lambda/lambda-translation.html | Lambda bodies become private methods of the enclosing class. | checked |

### Expressions, values and output

| Source | Anchor | What it established | Label |
|---|---|---|---|
| CEL language definition | github.com/google/cel-spec master `doc/langdef.md` §Overview, §Syntax, §Logical Operators, §Numbers, §JSON Data Conversion | Side-effect-free, no assignment; `&&` / `||` absorb errors; numbers compare across types; large ints map to JSON strings. | checked |
| CEL conformance tests | github.com/google/cel-spec `40a3c90` `tests/simple/testdata/*.textproto` | Shared test suite for two CEL implementations. | checked |
| cel-java | github.com/google/cel-java `d9cf3f98` `README.md:366`; `extensions/src/main/java/dev/cel/extensions/CelNativeTypesExtensions.java:361-381`, `:530-567` | Java 8+; POJO properties resolve to public getters first, public fields second. | checked |
| Datadog `Where`, signatures, limits, encoding, redaction | DataDog/dd-trace-java `1c373d58` `.../probe/Where.java:24-28`; `.../instrumentation/Types.java:290-375`; `.../debugger/Limits.java`; `.../util/MoshiSnapshotHelper.java:45-60`; `.../util/Redaction.java:19-170` | Structured location; signature conversion; default limits; typed-node encoding; keyword redaction. | checked |
| Datadog expression language | https://docs.datadoghq.com/dynamic_instrumentation/expression-language/ | No method calls; `@duration` in milliseconds. | checked |
| Arthas | https://arthas.aliyun.com/en/doc/watch.html; https://arthas.aliyun.com/en/doc/line.html; https://arthas.aliyun.com/en/doc/ognl-express.html | Watch points, subclass matching, line probes with locals, OGNL strict mode. | checked |
| BTrace | github.com/btraceio/btrace `develop` `docs/QuickReference.md` | Location kinds, `+Type` subtypes, `<init>`, `@TLS`. | checked |
| Byteman built-ins | github.com/bytemanproject/byteman main `docs/asciidoc/src/main/asciidoc/chapters/Byteman-Rule-Language-Standard-Built-Ins.adoc` | Flags, LinkMaps, recursive triggering. | checked |
| asciicast v2 | https://docs.asciinema.org/manual/asciicast/v2/ | Header line plus events; ignore unknown codes. | checked |
| jq precision | github.com/jqlang/jq master `NEWS.md` | Large integers compare correctly, arithmetic may truncate. | checked |
| JDK-8068730 | https://bugs.openjdk.org/browse/JDK-8068730 | Java 8 clock may be millisecond precision; fix version unverified. | checked (description only) |
| GNU `timeout` | https://www.gnu.org/software/coreutils/manual/html_node/timeout-invocation.html §Exit status | Wrapper exit-code convention. | checked |
| GNU Coding Standards §4.4 | https://www.gnu.org/prep/standards/html_node/Errors.html | `program:file:line: message` error format. | checked |
| Kubernetes required fields | https://kubernetes.io/docs/concepts/overview/working-with-objects/ §Required fields | Manifests require `apiVersion`. | checked |
| Docker Compose `version` | https://docs.docker.com/reference/compose-file/version-and-name/ | Top-level `version` is informative only. | checked |
| YAML 1.2.2 §7.3.3 | https://yaml.org/spec/1.2.2/ §7.3.3 | CEL in YAML needs quoting or block scalars. | unverified |
| New Relic 2024 State of the Java Ecosystem | https://newrelic.com/resources/report/2024-state-of-the-java-ecosystem | 35% of monitored apps on Java 17, 1.4% on Java 21. | checked |

### Relations between probes and Python

| Source | Anchor | What it established | Label |
|---|---|---|---|
| OpenTelemetry Context | https://opentelemetry.io/docs/specs/otel/context/ | Execution-scoped values across execution units. | checked |
| JFR `@Contextual` | https://docs.oracle.com/en/java/javase/25/docs/api/jdk.jfr/jdk/jfr/Contextual.html | Context for all events in the same thread during an event. | checked |
| DTrace speculative tracing | https://illumos.org/books/dtrace/chp-spec.html §13.1–13.7 | Record tentatively, commit or discard later. | checked |
| Sigma correlation rules | github.com/SigmaHQ/sigma-specification main `specification/sigma-correlation-rules-specification.md` | Separate correlation document with `group-by` and `timespan`. | checked |
| Elastic EQL | https://www.elastic.co/docs/reference/query-languages/eql/eql-syntax §Sequences, §Missing events | `sequence by ... with maxspan`; missing events need `maxspan`. | checked via search snippet only |
| Datadog span and trigger probes | dd-trace-java v1.67.1 `.../probe/SpanInstrumenter.java:40-150`, `TriggerProbe.java:113-157` | Span probe around a method; per-request debug sessions. | unverified |
| Lightrun Dynamic Traces | docs.lightrun.com | Single-thread trace entity. | unverified |
| Esper, Siddhi, Flink CEP, `MATCH_RECOGNIZE`, TraceQL, Tetragon | as reported by the correlation research track | Followed-by patterns with keys and windows. | unverified |
| `sys.monitoring` | https://docs.python.org/3/library/sys.monitoring.html | Events for every MVP point. | checked |
| Python 3.14 thread context | https://docs.python.org/3/using/cmdline.html | `thread_inherit_context` default. | checked |
| PEP 568 | https://peps.python.org/pep-0568/ | Deferred. | checked |
| asyncio, OTel Python, python-hunter, VizTracer, dd-trace-py | as reported by the Python research track | Task context copying; per-trace sessions. | unverified |

### Not reachable during research

- Elastic EQL syntax page: fetch timed out.
- Lightrun Dynamic Traces overview: HTTP 404.
- OpenJ9 `-XX:+EnableHCR` page: HTTP 404 (not needed: OpenJ9 `-p` is out of the MVP).
- Gradle and Maven docs on forked JVMs inheriting `JAVA_TOOL_OPTIONS`: not read.
- Per-hit overhead numbers: no primary source found; the MVP benchmark measures them.
