# Komar MVP findings

What the MVP measured and what the experiments showed. Numbers come from a MacBook (Apple silicon), JDK 25, unless noted.

## Verdict

The approach is feasible. All four probe kinds, CEL conditions and field-based capture work on JDK 11–25 with no changes to the program, and the realistic experiments found planted bugs from the JSONL alone. The main costs are startup (about 0.3–0.5 s per JVM) and per-hit cost on hot paths (0.3–1.4 µs per hit, most of it formatting JSON on the program's thread).

## Startup (per JVM)

| Run | Median |
|---|---|
| plain `java` | 0.03 s |
| Komar, empty spec | 0.29 s |
| Komar, probes without expressions | 0.36 s |
| Komar, probes with `when`/`what` | 0.50 s |

- The fixed cost is ByteBuddy loading (about 1,100 classes). cel-java is built lazily on a background thread.
- Several JVMs: a parent spawning 4 worker JVMs took 0.93 s with Komar vs 0.09 s plain, about 0.17 s per extra JVM.
- Class-data sharing did not help enough to be worth it.

## Cost per hit (O-8)

`bench/jvm/run.sh`: a tiny method called 2.4 M times in a loop, best of 3. Baseline 1.9 ns per call.

| Spec | Extra per call | Notes |
|---|---|---|
| no probes | 0 ns | |
| method entry, nothing captured | ~560 ns | |
| method entry, `when` always false | ~325 ns | CEL evaluation and scope setup, no record |
| method entry, `what: [args]` (object graph) | ~1,250 ns | value walk + JSON |
| method entry, `when` + 2 field expressions | ~1,400 ns | |
| line probe with one local | ~900 ns | |
| call-site probe on `Math.addExact` | ~880 ns | |
| `what: [args]` capped by default `max_hits` 1000 | ~120 ns | cost after the cap is the guard only |

- Profiling showed JSON formatting on the program thread dominated. Fixes applied: bigger initial buffer, no per-character escaping for plain strings, timestamp cached per millisecond, no activation copy per CEL evaluation. Together about −35% on capture paths.
- In a tight loop the agent queue overflows: about 80–90% of records were dropped in the benchmark, counted in `end.dropped`, and the program never blocked. This is the intended behavior (D-29), but on hot code users should rely on `when`, `max_hits` and `stop.hits`.
- Defaults (O-3): `max_hits` 1000 keeps a hot probe cheap; `classes` 50 was never reached in practice. `call_ms` is unused (opt-in calls not implemented).

## Agent jar size (O-8)

12.4 MB, relocated. Largest parts: guava 6.9 MB, ByteBuddy 4.8 MB (without its bundled ASM), protobuf 4.9 MB, cel-java 4.4 MB. guava and protobuf come from cel-java. Shrinking (ProGuard-style) was not tried; it is the obvious next step if size matters.

## cel-go vs cel-java (O-11)

93 expressions covering what Komar users write (paths, `has`, mixed comparisons, arithmetic rules, strings, indexing, maps, macros, ternary, short-circuit, conversions, timestamps).

- Compile/check: identical accept/reject on all 93. The pre-launch check never accepts what the agent rejects, or the other way round.
- Evaluation: same value-or-error outcome on all 93. Only formatting differs: `string(1500.0)` is `"1500"` in cel-go and `"1500.0"` in cel-java.
- Error texts differ between the engines; `@error` strings must not be treated as stable.
- `has()` on a null field of a program object is false (Komar's value layer treats null fields as absent), but true for a null map value. This makes `has(x.customer) && x.customer.vip` a working null guard; it should be documented as intended.

## DuckDB and jq (O-9)

- `read_json_auto` types `what` as one STRUCT with the union of keys across probes; missing keys are NULL. A key whose type differs between records becomes JSON. With hundreds of distinct keys DuckDB switches to `MAP(VARCHAR, …)`. The type depends on the data.
- Robust recipe: declare `what` as JSON and use `what->>'$.key'`:
  `SELECT what->>'$.tier' FROM read_json('out.jsonl', columns={'kind':'VARCHAR','probe':'VARCHAR','point':'VARCHAR','call':'BIGINT','pid':'BIGINT','what':'JSON'}) WHERE kind='hit'`
- jq works well: `jq -r 'select(.kind=="hit")|.probe' out.jsonl | sort | uniq -c`; `jq -c 'select(.kind=="diag" and .level!="info")|{probe,level,msg}' out.jsonl`.

## Experiments

- **HTTP service bug hunt** (JDK HttpServer → service → repository): a spec with method, `when`, line and call-site probes worked on the first try and found both planted bugs (wrong tier casing, NPE for an unknown customer) with two jq queries. 87 records, 24 KB.
- **Concurrency** (thread pool, CompletableFuture, lambdas): call trees correct per thread, lambdas get their own call ids, `stop.hits` exact under 8 spinning threads. Method references and lambdas handed straight to an executor start new trees (`parent: null`), as designed.
- **Multi-JVM**: one agent per JVM, distinct pids, exit code propagated, unplaced probe reported once.

## Problems found, not fixed yet

1. `class: PricingService` without its package reports "no class matched … (none was loaded)" although `shop.PricingService` was loaded. Should hint at the qualified name.
2. "no parameter named customer" should list the available names.
3. `end.dropped` lumps agent queue overflow with hits arriving after `stop.hits`; a reader cannot tell them apart.
4. `stop.hits: 1000` coincides with the default `max_hits` and prints a misleading "max_hits reached" warning.
5. A probe whose only `what` item is left out still writes hits with `"what":{}`.
6. Startup and per-hit cost are dominated by ByteBuddy and JSON formatting; moving formatting off the program thread and plain ASM instead of ByteBuddy are the next levers.

## Not covered

`-p` attach, opt-in method calls (`calls: true`), calls to constructors, `invokedynamic` call targets, the `jvm` filter, Java 8 runtime, OpenJ9, containers, Windows.
