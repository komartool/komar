# The Komar spec language

A Komar spec is a YAML file that tells `komar run` where to observe a JVM program, when to record,
and what to capture. This document describes the language as decided for the MVP. Every rule
cites the decision it comes from as `(D-n)`; the decisions themselves are summarised in
[decisions.md](decisions.md).

Related documents:

- [README.md](README.md) — the overview of Komar's design.
- [output-format.md](output-format.md) — the JSONL records a spec produces.
- [jvm-runtime.md](jvm-runtime.md) — how the agent places probes in the JVM.
- [lifecycle-and-failures.md](lifecycle-and-failures.md) — sessions, `stop`, errors and exit codes.
- [scope-and-roadmap.md](scope-and-roadmap.md) — what is out of the MVP, and the open questions.

Words used throughout:

| Word | Meaning |
|---|---|
| probe | one entry in `probes:` — a location, a condition and a list of captures |
| point | a moment a probe can observe: `entry`, `exit`, `exception`, a source line, `before`, `after` |
| hit | one occurrence of a point at run time; each hit that passes `when` becomes one record |
| place | install a probe's bytecode into a loaded class |
| capture | evaluate a `what` item and write its value into the hit record |
| session | one `komar run` invocation, from start until the end record |

Proposals that the person has not confirmed are marked **proposal, not confirmed** and listed in
[scope-and-roadmap.md#open-questions](scope-and-roadmap.md#open-questions).

---

## 1. File shape

The spec file is a YAML map. Its only required key is `probes:`. The other top-level keys apply
to the whole spec (D-14).

```yaml
version: 1          # optional
calls: false        # optional
limits: { depth: 3 }  # optional
stop: { after: 60s }  # optional
jvm: { main: 'com.shop.App' }  # optional
probes:
  - id: charge
    where: { class: com.shop.Billing, method: charge, at: [exit] }
    what: [ret]
```

| Key | Type | Default | Meaning | Decision |
|---|---|---|---|---|
| `version` | integer | the version the CLI implements | version of the spec language. **Proposal, not confirmed.** | D-46 |
| `calls` | boolean | `false` | allows expressions to call methods of the target program | D-13, D-14 |
| `limits` | map | see [§1.1](#11-limits) | capture and volume limits for every probe | D-23, D-31, D-34, D-44 |
| `stop` | map | none: the session ends only on program exit or a signal | conditions that end the session | D-28 |
| `jvm` | map | every JVM | records only in JVMs that match | D-27 |
| `probes` | list | — (required) | the probes | D-5 |

An unknown top-level key is a spec error (D-33). This is also why the optional `version` is enough:
a spec written for a newer format fails loudly instead of being misread (D-46, proposal).

### 1.1 `limits`

`limits` holds the capture limits and the volume limits. Komar has built-in defaults. The top-level
`limits` map overrides them for the whole spec. A probe can carry its own `limits` map that
overrides them for that probe only (D-23).

| Key | Default | Meaning | Decision |
|---|---|---|---|
| `depth` | 3 | how many levels of object fields a capture follows | D-23 |
| `items` | 100 | how many elements of a collection or array a capture writes | D-23 |
| `string` | 1,000 | how many characters of a string a capture writes | D-23 |
| `fields` | 50 | how many fields of one object a capture writes | D-23 |
| `max_hits` | 1,000 | hits after which a probe stops firing. **Default is a proposal, not confirmed.** | D-34, D-44 |
| `classes` | 50 | classes one `where` may match. **Default is a proposal, not confirmed.** | D-34 |
| `call_ms` | 50 | time after which a slow method call disables calls for the probe. **Default is a proposal, not confirmed.** | D-31 |

The person confirmed the first four defaults. The last three numbers are proposals and have not
been measured; the MVP benchmark is meant to check them (D-41).

A per-probe `limits` map replaces only the keys it names. `limits: { depth: 5 }` on a probe
changes the depth and keeps every other limit. **This per-key merge is a proposal, not confirmed**
(D-23).

```yaml
limits: { depth: 3, items: 100, string: 1000, fields: 50, max_hits: 1000, classes: 50, call_ms: 50 }
probes:
  - id: deep_order
    where: { class: com.shop.Billing, method: charge, at: [entry] }
    limits: { depth: 5, max_hits: 50 }   # this probe only
    what: [args]
```

A probe that reaches `max_hits` stops firing and reports it once (D-34). A `where` that matches
more classes than `classes` is a spec error before launch, and the message shows the count.
Values cut by `depth`, `items`, `string` or `fields` stay visible in the output through `@`
markers; see [output-format.md](output-format.md).

The per-probe cap is called `max_hits` so that it cannot be confused with `stop.hits`, which ends
the whole session. `hits` inside `limits` and `max_hits` inside `stop` are spec errors with a
suggestion (D-44).

### 1.2 `stop`

`stop` ends the session at the first condition reached. `after` is a duration, and `hits` counts
hits across all probes (D-28).

```yaml
stop: { after: 60s, hits: 500 }
```

In launch mode a fired `stop` also terminates the program that Komar started. In attach mode it
ends observation, and the target keeps running. The details are in
[lifecycle-and-failures.md](lifecycle-and-failures.md).

### 1.3 `jvm`

A command such as `./gradlew test` starts several JVMs, and each of them loads the agent. By
default every JVM records. `jvm` makes JVMs that do not match stay idle (D-27).

```yaml
jvm: { main: 'com.shop.App' }
```

The MVP matches on the main class. Other match keys are not decided
([open questions](scope-and-roadmap.md#open-questions)).

---

## 2. Probes

A probe has four keys and one optional key.

| Key | Required | Meaning |
|---|---|---|
| `id` | yes | the probe's name; every hit record carries it |
| `where` | yes | where the probe is placed and which points it observes |
| `when` | no | a CEL condition; a hit is recorded only when it is true |
| `what` | no | the list of values to capture at each hit |
| `limits` | no | per-probe overrides of the limits ([§1.1](#11-limits)) |

```yaml
probes:
  - id: charge
    where: { class: com.shop.Billing, method: charge, at: [entry, exit, exception] }
    when: 'vars.order.total > 1000'
    what:
      - args
      - ret
```

One probe can select several points of one method. Each point writes its own record when it
happens, not when the call finishes. All records of one invocation share a call id, so analysis
can pair an entry with its exit (D-5).

---

## 3. Three kinds of probes

`where` is a structured map with separate keys, not one combined string (D-2). The keys present
decide which of three kinds the probe is (D-8):

| Kind | Recognised by | Allowed `at` values | What it observes |
|---|---|---|---|
| method probe | neither `line` nor `call` | `entry`, `exit`, `exception` | the start, normal end and exceptional end of a method (D-2, D-5) |
| line probe | `line:` | none — `at` is not allowed | each execution of the listed source lines (D-2, D-8) |
| call-site probe | `call:` | `before`, `after`, `exception` | one call made from inside the method (D-2, D-9) |

A probe has exactly one kind. To observe a method's entry and one of its lines, write two probes.

### 3.1 Method probe

```yaml
- id: charge
  where:
    class: com.shop.Billing
    method: charge
    at: [entry, exit, exception]
  what:
    - args
    - ret                        # captured at exit only
    - error: exception.message   # captured at exception only
```

- `entry` fires when the method starts.
- `exit` fires when it returns normally.
- `exception` fires when it ends by throwing. Komar records the exception and lets it continue
  unchanged.

### 3.2 Line probe

```yaml
- id: loop
  where: { class: com.shop.Billing, method: charge, line: [51, 57] }
  what:
    - i: vars.i
```

A line probe fires each time execution reaches one of the listed lines. The class needs line
number information; javac writes it by default. Local variable names additionally need the class
to be compiled with `-g` or `-g:vars` (D-19).

### 3.3 Call-site probe

A call-site probe observes one call made inside the probed method. Komar changes only the calling
method, so the called method can be JDK or library code that Komar does not touch (D-2).

```yaml
- id: gw
  where:
    class: com.shop.Billing
    method: charge
    call: { class: com.pay.PaymentGateway, subtypes: true, method: send }
    at: [before, after, exception]
  what:
    - order: call.args[0]
    - gateway: call.target
    - ret
```

`before`, `after` and `exception` mean the same as `entry`, `exit` and `exception` on a method
probe. `after` fires only on a normal return and never carries an exception. `exception` fires
only when the call throws and never carries a return value. The original exception is rethrown
unchanged (D-9).

---

## 4. Matching in `where`

### 4.1 Classes

`class` is an exact fully qualified name or a wildcard pattern (D-6).

```yaml
where: { class: com.shop.Billing, method: charge, at: [entry] }
where: { class: 'com.shop.*Service', method: '*', at: [exception] }
```

Subtypes are matched only on request. With `subtypes: true` the probe covers every class that
extends or implements the named type, including classes that load later.

```yaml
where: { class: com.pay.PaymentGateway, subtypes: true, method: send, at: [entry, exit] }
```

A broad pattern is capped by `limits.classes` ([§1.1](#11-limits)). Whether regular expressions are
also allowed is not decided ([open questions](scope-and-roadmap.md#open-questions)).

### 4.2 Methods and overloads

`method` is a method name. Without `params`, the probe covers every overload with that name.
`params` narrows it to the overloads whose parameter types match, written in Java source syntax
(D-7).

```yaml
where: { class: com.pay.Gateway, method: send, at: [entry] }                         # all overloads
where: { class: com.pay.Gateway, method: send, params: [String, int], at: [entry] }  # one overload
```

Rules for the types in `params` (D-7 and its follow-ups):

| Written as | Matches |
|---|---|
| a name with a dot, `java.sql.Date` | exactly that fully qualified type |
| a bare name, `Date` | any type whose simple name is `Date`; if two overloads match, both are probed |
| a primitive or array, `int`, `byte[]` | exactly as written |
| a generic type, `'List<Order>'` | any `List`; type arguments are ignored because they do not exist at run time |
| varargs, `Object...` | the same as `Object[]` |
| a type variable, `K` | matched through the method's generic signature in the class file |

```yaml
# source: public V put(K key, V value)
where: { class: com.shop.Cache, method: put, params: [K, V], at: [entry] }
# source: void log(String fmt, Object... args)
where: { class: com.shop.Log, method: log, params: [String, Object...], at: [entry] }
```

The rule throughout is "write what you see in source". A name that matches nothing is reported
instead of passing silently. Obfuscated builds lose the generic signature, line numbers and local
names, and their classes are renamed, so they cannot be matched by source names at all. Each hit
record names the overload that actually ran.

### 4.3 Constructors

A constructor is written as `method: new`. `params` selects among overloaded constructors
(D-11, D-42).

```yaml
- id: created
  where: { class: com.shop.Order, method: new, params: [long], at: [exit] }
  what: [this]
```

At constructor `entry` the object does not exist yet, so `this` is not available there. It is
available at `exit` and `exception` (D-11).

Accepting the JVM name `<init>` as an alias for `new` is a **proposal, not confirmed** (D-42).
Static initializers are not probe targets in the MVP (D-11).

### 4.4 Lambdas and anonymous classes

Agents copy line numbers from source, and a line inside a lambda looks like part of the method.
Line probes and call-site probes on method `M` therefore cover (D-12):

- `M`'s own code;
- lambda bodies created in `M`;
- methods of anonymous and local classes declared in `M`, including classes that load later.

```yaml
# line 51 is inside orders.forEach(o -> { ... }) in Billing.charge
- id: per_order
  where: { class: com.shop.Billing, method: charge, line: [51] }
  what:
    - id: vars.o.id
```

The probe fires each time the lambda body reaches line 51, possibly later or on another thread.
Each record names both the method that actually ran and the method named in the spec. Method
probes (`at: entry/exit/exception`) still target `M` only; this part is a reading of D-12 that was
not asked separately.

### 4.5 Call targets

The class in `call:` is matched against the run-time class of the object being called, using the
same rules as [§4.1](#41-classes). A static call is matched against the class it names (D-10).

```java
class Billing {
    private PaymentGateway gateway;          // holds a StripeGateway at run time
    void charge(Order order) { gateway.send(order); }
}
```

```yaml
call: { class: com.pay.StripeGateway, method: send }                    # only Stripe
call: { class: com.pay.PaymentGateway, subtypes: true, method: send }   # any gateway
```

Naming an interface or an abstract class without `subtypes: true` can never match a run-time
class, so Komar warns about it when it reads the spec (D-10).

---

## 5. Expressions: `when` and `what`

`when` and every `what` item are CEL expressions. The CLI parses and type-checks all of them with
cel-go before it launches or attaches to anything, so a mistake never starts the program (D-15).
The agent evaluates them inside the target JVM on live objects. CEL itself has no assignment and
no side effects; method calls are the only way an expression can run program code, and they are
off by default ([§5.5](#55-reading-values-and-method-calls)).

### 5.1 `what`

`what` is a list. Each item is either a bare expression, keyed by its own text, or a one-entry
map `name: expression` (D-16).

```yaml
what:
  - args                        # key "args"
  - ret                         # key "ret"
  - customer: vars.order.customer
  - total: vars.order.total
```

The captured values go under the record's `what` object, key for key:

```jsonl
{"kind":"hit","probe":"charge","point":"exit","what":{"args":[...],"ret":true,"customer":{...},"total":1500.0}}
```

Use a bare item for a simple name and give a name to anything with a path or an operator. Two
items with the same output key in one probe are a spec error; **this is a proposal, not
confirmed** (D-16).

### 5.2 Variables

These variables are visible to expressions. Top-level names belong to Komar. Program names
(parameters and local variables) are reachable only under `vars.`, so they can never collide with
Komar's names: a parameter called `call` is `vars.call`, and a field called `call` is
`this.call` (D-19, D-20).

| Variable | Available at | Meaning | Decision |
|---|---|---|---|
| `this` | every point, except constructor `entry` | the object running the probed method (static methods have none) | D-11, D-20 |
| `args` | every point | the probed method's arguments by position, `args[0]`, `args[1]`, … | D-19 |
| `vars.<name>` | every point | a parameter or local variable by its source name | D-19 |
| `ret` | `exit`, `after` | the return value: of the method at `exit`, of the call at `after` | D-17, D-20 |
| `exception` | `exception` | the thrown exception: of the method, or of the call at a call site | D-9, D-20 |
| `duration` | `exit`, `exception`, `after` | milliseconds since the method's (or call's) entry, as a decimal number | D-43 |
| `call.target` | call-site probes | the object whose method is called; absent for static calls | D-20 |
| `call.args` | call-site probes | the arguments passed in the call | D-20 |

`this`, `args` and `vars` always mean the probed method, in every kind of probe. At a call-site
probe the call has its own names under `call.` (D-20):

| Expression at `gateway.send(order)` inside `Billing.charge` | Value |
|---|---|
| `this` | the `Billing` object |
| `args` | `[order]` — the arguments of `charge` |
| `call.target` | the gateway, for example a `StripeGateway` |
| `call.args` | `[order]` — the arguments of `send` |
| `ret` at `after` | the result of `send` |

`args` and `call.args` look the same here only because `charge` passes its own `order` along.
For `gateway.send(order.payment(), 3)`, `call.args` is `[payment, 3]`.

`duration` is a plain number of milliseconds, so `duration > 100` and `duration > 0.5` both work.
CEL compares integers and doubles on one number line, so `100` and `100.0` mean the same (D-43).

```yaml
- id: slow_charge
  where: { class: com.shop.Billing, method: charge, at: [exit] }
  when: 'duration > 100'
  what:
    - ms: duration
```

### 5.3 Names from source: `vars`

`args[n]` always works. `vars.<name>` gives a parameter or local variable by the name in the
source, but only when the class was compiled with names (D-19):

| Name | Needs |
|---|---|
| `args[0]` | nothing |
| `vars.order` (parameter) | javac `-parameters` or `-g` |
| `vars.i` (local variable) | javac `-g` or `-g:vars` |

The CLI cannot see the program's classes, so it accepts any name after `vars.`. A misspelled or
unavailable name shows up only when the agent places the probe. It is then handled by the rules
for missing values below, and Komar reports why the name is missing, for example "compiled
without `-g`".

```yaml
- id: big_orders
  where: { class: com.shop.Billing, method: charge, at: [entry] }
  when: 'vars.order.total > 1000'
  what:
    - args
    - customer: vars.order.customer
```

### 5.4 Values missing at a point

One probe can select points where some variables do not exist, for example `ret` at `entry`. The
rules are forgiving by design; the person's principle is "I like the forgiving design for AI
agents" (D-17).

- **In `what`**, an item whose variables do not exist at a point is not captured there. Its key
  is left out of that record. A key present with `null` always means the value really was null
  (D-17).
- **In `when`**, a condition that uses a variable missing at a point is false there, so that point
  never fires. The CLI prints one warning before launch (D-18).

```yaml
- id: charge
  where: { class: com.shop.Billing, method: charge, at: [entry, exit, exception] }
  what:
    - args
    - ret
    - error: exception.message
```

```jsonl
{"kind":"hit","probe":"charge","point":"entry","what":{"args":[...]}}
{"kind":"hit","probe":"charge","point":"exit","what":{"args":[...],"ret":{...}}}
{"kind":"hit","probe":"charge","point":"exception","what":{"args":[...],"error":"card declined"}}
```

```yaml
- id: null_returns
  where: { class: com.shop.Billing, method: charge, at: [entry, exit] }
  when: 'ret == null'     # entry can never fire: ret does not exist there
```

```text
komar: warning: spec.yaml:<line>:<col>: probe "null_returns": when uses ret, not available at entry; entry will never fire
```

A `when` that fails while the program runs, for example on a null field, counts as false. That is
a run-time failure, described in [lifecycle-and-failures.md](lifecycle-and-failures.md) (D-30).
CEL's `&&` and `||` absorb errors when the other side decides the result, so
`has(vars.order.customer) && vars.order.customer.vip` never fails.

### 5.5 Reading values and method calls

By default, observing never runs the program's code. Expressions and captures read values
straight from fields, private fields included (D-13):

- `x.total` reads the field `total`. It never calls `getTotal()`.
- Strings, boxed numbers, `BigDecimal`, `java.time` types, enums, lists, sets, maps and arrays are
  rendered as plain values, but only when the object's class is exactly the JDK class.
- Any other object is written as a map of its fields, within the limits of [§1.1](#11-limits).

`calls: true` at the top of the spec allows expressions to call methods, such as `x.getTotal()`
or `x.toString()`. Without it, any method call in any `when` or `what` is a spec error that names
the flag (D-14).

```yaml
calls: true
probes:
  - id: charge
    where: { class: com.shop.Billing, method: charge, at: [entry] }
    when: 'args[0].getTotal() > 1000'
    what:
      - customer: args[0].customer          # still a field read
      - order: args[0].toString()           # a method call
```

`calls` is one flag for the whole spec. When it is on, every probe may call methods. Records
whose values came from a call are marked as such ([output-format.md](output-format.md)). The rule
this follows is: Komar never changes program code, and observation runs program code only when
the spec asks for it (D-13).

Calls are contained by guardrails (D-31):

- an exception thrown by a called method stays inside the agent and becomes an `@error` value;
- probes do not fire while Komar evaluates an expression, so a probed getter does not record
  Komar's own calls;
- after a probe's first call slower than `limits.call_ms`, Komar stops making calls for that probe
  and reports it once.

Side effects a called method performs are not undone; that is the cost of `calls: true`. Details
are in [lifecycle-and-failures.md](lifecycle-and-failures.md).

### 5.6 Writing CEL inside YAML

CEL uses characters that YAML treats specially: `!`, `[`, `{`, `?`, `:`, `#` and quotes. Quote
every expression, or use a block style. YAML 1.2.2 §7.3.3 forbids `: ` and ` #` inside plain
scalars and forbids starting them with indicators such as `!`; this reading of the YAML
specification is unverified.

```yaml
when: '!has(vars.order.customer)'         # quoted: a bare ! would be read as a YAML tag
what:
  - first: 'args[0]'                      # quoted inside a flow context
  - label: "vars.order.total > 100 ? 'big' : 'small'"
when: |
  vars.order.total > 1000 &&
  vars.order.customer.vip
```

In a block list (`- item` on separate lines) most expressions work without quotes. Inside an
inline `[...]` list, quote anything that contains brackets.

---

## 6. Full example

The spec assembled from all MVP decisions:

```yaml
version: 1                                 # D-46: optional; proposal, not confirmed
calls: false                               # D-14: method calls in expressions are off
limits: { depth: 3, items: 100, string: 1000, fields: 50, max_hits: 1000, classes: 50, call_ms: 50 }
stop: { after: 60s, hits: 500 }            # D-28: optional; in launch mode also ends the program
jvm: { main: 'com.shop.App' }              # D-27: optional; default is every JVM

probes:
  - id: charge                             # method probe
    where:
      class: com.shop.Billing
      method: charge
      params: [Order]
      at: [entry, exit, exception]
    when: 'vars.order.total > 1000'
    what:
      - args
      - ret                                # only at exit (D-17)
      - error: exception.message           # only at exception
      - ms: duration                       # milliseconds (D-43)

  - id: loop                               # line probe
    where: { class: com.shop.Billing, method: charge, line: [51, 57] }
    what:
      - i: vars.i

  - id: gw                                 # call-site probe
    where:
      class: com.shop.Billing
      method: charge
      call: { class: com.pay.PaymentGateway, subtypes: true, method: send }
      at: [before, after, exception]
    what:
      - order: call.args[0]
      - gateway: call.target

  - id: created                            # constructor (D-42)
    where: { class: com.shop.Order, method: new, at: [exit] }
    what: [this]
```

Run it at launch or against a running JVM:

```text
komar run spec.yaml -o out.jsonl -- java -jar app.jar
komar run spec.yaml -o out.jsonl -p 4242
```

---

## 7. Common mistakes and what Komar says

The CLI checks the whole spec before it starts anything. It reports every problem at once on
stderr, in the compiler format `komar: <file>:<line>:<col>: <message>`, with a suggestion when one
is obvious. It then prints a count line and exits with code 2, without launching, attaching or
touching the `-o` file (D-33). Warnings use the same format and do not stop the session; prefixing
them with `warning:` is a **proposal, not confirmed** (D-33).

The exact wording of each message is not fixed. The examples show the kind of message, with
placeholder positions. Rows marked \* follow from the decisions but were not asked as separate
questions.

| Mistake | Example | What Komar does | Decision |
|---|---|---|---|
| unknown key | `wher:` | error: `unknown key "wher" (did you mean "where"?)` | D-33 |
| `at` on a line probe \* | `line: [51]` with `at: [entry]` | error: `at` is not allowed on a line probe | D-8 |
| point from the wrong kind \* | `at: [before]` on a method probe | error: allowed values are `entry`, `exit`, `exception` | D-8, D-9 |
| method call without the flag | `when: 'args[0].getTotal() > 1000'` | error: method call requires `calls: true` at the top of the spec | D-14 |
| `ret` in `when` at `entry` | `when: 'ret == null'` with `at: [entry, exit]` | warning: `entry` will never fire | D-18 |
| `ret` in `what` at `entry` | `what: [ret]` with `at: [entry, exit]` | no message; the key is left out of entry records | D-17 |
| `this` at constructor entry | `method: new`, `at: [entry]`, `what: [this]` | not available there; the exact handling (error or omission) is not decided | D-11 |
| per-probe cap in the wrong section | `limits: { hits: 50 }` | error: `unknown key "hits" (did you mean "max_hits"?)` | D-44 |
| run total in the wrong section | `stop: { max_hits: 500 }` | error with a suggestion of `hits` | D-44 |
| interface without `subtypes` in `call` | `call: { class: com.pay.PaymentGateway, method: send }` | warning: an interface can never be the run-time class | D-10 |
| pattern matching too many classes | `class: 'com.*'` | error with the number of matched classes | D-34 |
| CEL type error \* | `when: 'duration > "100"'` | error from the CEL type check | D-15 |
| misspelled `vars` name | `vars.ordr.total` | accepted at check time; reported when the probe is placed, then treated as missing | D-19 |

```text
$ komar run spec.yaml -o out.jsonl -- java -jar app.jar
komar: spec.yaml:12:11: unknown key "wher" (did you mean "where"?)
komar: spec.yaml:17:7: at is not allowed on a line probe (probe "loop")
komar: spec.yaml:24:9: what: method call "getTotal()" requires "calls: true" at the top of the spec
komar: 3 errors, nothing started
$ echo $?
2
```

Exit code numbers are working values ([lifecycle-and-failures.md](lifecycle-and-failures.md),
D-32).

---

## 8. Not in the language for the MVP

These are postponed, not rejected; [scope-and-roadmap.md](scope-and-roadmap.md) lists them with
their reasons.

- Field read and write probes, and throw or catch inside a method (D-2).
- Static initializers (D-11).
- Conditions relative to other probes, such as spans (D-21). In the MVP, records are related
  afterwards through `call`, `parent` and `thread` ids in the output, and across threads through a
  captured program value such as a request id ([output-format.md](output-format.md)).
- Stack capture (D-21) and rate limits (D-34).
- Redaction of secrets. Komar captures values as they are, so passwords and tokens in captured
  values reach the output file (D-35).

---

## 9. Forward compatibility with Python

Python is not in the MVP, but the language was shaped so that one spec format can later serve
CPython. These notes come from the research and are not verified in a Python implementation.

| Feature | JVM | Python, later |
|---|---|---|
| `class` + `method` | fully qualified class and method name | module plus qualified name, e.g. `Outer.method.<locals>.inner` |
| `params` | selects overloads (D-7) | Python has no overloads; the key would be ignored or rejected |
| `method: new` | constructor (D-42) | could map to `__init__` (D-11) |
| line and call-site probes | bytecode insertion | `sys.monitoring` `LINE`, `CALL` events |
| `vars.<name>` | needs `-parameters` / `-g` (D-19) | names are always available |
| `call.args` | all call arguments (D-20) | `CALL` passes only the callable and the first argument; the rest may come only from the callee's entry |
| reading values | fields, exact JDK types (D-13) | `__dict__` / `__slots__`; properties and `__repr__` run code and would count as calls |
| same-thread relations | thread (D-21) | asyncio tasks share threads, so relations would follow tasks |

Keeping program names under `vars.`, matching call targets by run-time class, and the
`before` / `after` / `exception` words were chosen partly so that they mean the same in both
runtimes (D-10, D-19, D-20).
