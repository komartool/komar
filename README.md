# Komar

Observe a running JVM program without changing its code. You describe what to capture in a YAML spec, Komar instruments the program at load time and writes one JSONL file you can query with jq or DuckDB. Built to be driven by AI agents as much as by people.

Status: MVP. Launch mode only, JVM only.

## Build

Needs Go 1.24+ and JDK 21+ (to build the agent). The program you observe can run on Java 8+.

```sh
make build        # bin/komar, agent jar embedded
```

## Use

```sh
komar run spec.yaml -o out -- java -cp app.jar com.shop.Main
```

```yaml
stop: { after: 30s }
probes:
  - id: big_orders
    where: { class: com.shop.OrderService, method: charge, at: [entry, exit] }
    when: 'vars.order.total > 1000'
    what:
      - id: vars.order.id
      - customer: vars.order.customer.name
      - ret

  - id: loop
    where: { class: com.shop.OrderService, method: charge, line: [42] }
    what: [vars.item, vars.subtotal]

  - id: repo_get
    where:
      class: com.shop.OrderService
      method: charge
      call: { class: java.util.HashMap, method: get }
      at: [after]
    what: [call.args, ret]
```

- Probe kinds: method and constructor (`entry`, `exit`, `exception`), line, and call site (`before`, `after`, `exception`).
- `when` and `what` are [CEL](https://cel.dev) expressions over `this`, `args`, `vars`, `ret`, `exception`, `duration` and `call`. Values are read from fields; program methods never run.
- `vars` needs `-parameters` or `-g`; locals at a line need `-g`.
- The spec is checked before launch; every error is reported with its line and column.

`-o out` writes two files; `-f jsonl` or `-f view` keeps only one:

- `out.view`: a compact call tree for reading. Repeated values and identical executions are shown once.
- `out.jsonl`: the full record, one JSON line per record: `run`, then `hit` and `diag`, then `end`.

```sh
jq -c 'select(.probe=="big_orders" and .point=="exit") | .what' out.jsonl
```

The formats are described in [docs/design/output-format.md](docs/design/output-format.md).

## Develop

```sh
make lint         # Go and Java linters, architecture rules
make check        # lint + unit and e2e tests
```

E2E cases live in `e2e/testdata/*.txtar`. Design docs are in `docs/design/`, MVP results in `docs/mvp-findings.md`.
