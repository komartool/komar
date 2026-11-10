package compact

import (
	"strings"
	"testing"
)

const head = legend + "\n"

func TestRender(t *testing.T) {
	cases := []struct {
		name   string
		probes []string
		dump   string
		want   string
	}{
		{
			name:   "object cut by the fields limit keeps the captured fields",
			probes: []string{"p"},
			dump: `{"kind":"hit","probe":"p","point":"entry","method":"Svc.run(Configuration)","call":1,"parent":null,"pid":1,"what":{"args":[{"@type":"org.x.Configuration","lazy":false,"cacheEnabled":true,"@truncated":["a","b"]}]}}
{"kind":"hit","probe":"p","point":"entry","method":"Svc.run(Configuration)","call":2,"parent":null,"pid":1,"what":{"args":[{"@type":"org.x.Configuration","@truncated":true}]}}
{"kind":"end","reason":"exit","code":0}`,
			want: head + `end: reason=exit code=0
hits: p@entry 2
executions: 2, distinct: 2

Svc.run(Configuration)  args=(org.x.Configuration#1{lazy: false, cacheEnabled: true, …})

Svc.run(Configuration)  args=(Configuration{…})
`,
		},
		{
			name:   "a type is named in full on first use and when its simple name clashes",
			probes: []string{"p"},
			dump: `{"kind":"hit","probe":"p","point":"entry","method":"com.shop.Svc.put(Item)","call":1,"parent":null,"pid":1,"what":{"args":[{"@type":"com.a.Item","id":1}]}}
{"kind":"hit","probe":"p","point":"entry","method":"com.shop.Svc.put(Item)","call":2,"parent":null,"pid":1,"what":{"args":[{"@type":"com.b.Item","id":2}]}}
{"kind":"hit","probe":"p","point":"entry","method":"com.shop.Svc.put(Item)","call":3,"parent":null,"pid":1,"what":{"args":[{"@type":"com.a.Item","id":3}]}}
{"kind":"hit","probe":"p","point":"entry","method":"com.shop.Svc.put(Item)","call":4,"parent":null,"pid":1,"what":{"args":[{"@type":"com.b.Item","id":4}]}}
{"kind":"end","reason":"exit","code":0}`,
			want: head + `end: reason=exit code=0
hits: p@entry 4
executions: 4, distinct: 4

com.shop.Svc.put(Item)  args=(com.a.Item{id: 1})

Svc.put(Item)  args=(com.b.Item{id: 2})

Svc.put(Item)  args=(Item{id: 3})

Svc.put(Item)  args=(com.b.Item{id: 4})
`,
		},
		{
			name:   "outcome first, repeated diagnostics folded, probes without hits named",
			probes: []string{"target", "helper"},
			dump: `{"kind":"run","format":1,"spec":"s.yaml","cmd":["mvn","test"]}
{"kind":"diag","level":"info","msg":"agent started","pid":1}
{"kind":"diag","level":"error","msg":"could not instrument org.m.A: boom","class":"org.m.A","pid":1}
{"kind":"diag","level":"error","msg":"could not instrument org.m.B: boom","class":"org.m.B","pid":1}
{"kind":"diag","level":"error","msg":"could not instrument org.m.C: boom","class":"org.m.C","pid":2}
{"kind":"diag","level":"warn","probe":"target","msg":"max_hits reached; probe stopped","pid":1}
{"kind":"hit","probe":"target","point":"entry","method":"Svc.run()","call":1,"parent":null,"pid":1,"what":{}}
{"kind":"end","reason":"stop","code":null,"dropped":3}`,
			want: head + `end: reason=stop dropped=3
no hits: helper (the run was stopped before they fired)
hits: target@entry 1
error: could not instrument org.m.A: boom   ×3 classes, also org.m.B, org.m.C
warn: target max_hits reached; probe stopped
executions: 1, distinct: 1

Svc.run()
`,
		},
		{
			name:   "proxies are shown by the class they stand for",
			probes: []string{"p"},
			dump: `{"kind":"hit","probe":"p","point":"entry","method":"Svc.run(Tariffs, Repo, Clock, Calc)","call":1,"parent":null,"pid":1,"what":{"args":[{"@type":"com.shop.Tariffs$$EnhancerBySpringCGLIB$$1a2b","cache":{"@type":"java.util.HashMap"},"rates":[1,2,3]},{"@type":"com.sun.proxy.$Proxy12","h":{"@type":"Handler","x":1}},{"@type":"com.shop.Clock$MockitoMock$123","mockitoInterceptor":{"@type":"I"}},{"@type":"com.shop.$Calc$Definition$Intercepted","x":1}]}}
{"kind":"end","reason":"exit","code":0}`,
			want: head + `end: reason=exit code=0
hits: p@entry 1
executions: 1, distinct: 1

Svc.run(Tariffs, Repo, Clock, Calc)  args=(com.shop.Tariffs{proxy}, com.sun.proxy.$Proxy12{proxy}, com.shop.Clock{proxy}, com.shop.Calc{proxy})
`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Render([]byte(c.dump), c.probes)
			if got != c.want {
				t.Errorf("view differs\n--- got\n%s\n--- want\n%s", got, c.want)
			}
		})
	}
}

func TestRenderContents(t *testing.T) {
	var lines []string
	for i, n := range []string{"1", "2", "3"} {
		id := strings.Repeat("x", i+1)
		lines = append(lines,
			`{"kind":"hit","probe":"p","point":"entry","method":"Svc.run(int)","call":"`+id+`","parent":null,"pid":1,"what":{"args":[`+n+`]}}`,
			`{"kind":"hit","probe":"p","point":"exit","method":"Svc.run(int)","call":"`+id+`","parent":null,"pid":1,"what":{"ret":`+n+`0}}`)
	}
	lines = append(lines, `{"kind":"end","reason":"exit","code":0}`)
	got := render([]byte(strings.Join(lines, "\n")), []string{"p"}, layout{contentsLines: 1, contentsEntries: 2})
	want := head + `end: reason=exit code=0
hits: p@entry 3, p@exit 3
executions: 3, distinct: 3
contents (line numbers in this file):
  [1] L9 Svc.run(int)  args=(1) ⇒ 10
  [2] L12 Svc.run(int)  args=(2) ⇒ 20
  … 1 more, lines 15–17
[1]
Svc.run(int)  args=(1)
  ⇒ 10
[2]
Svc.run(int)  args=(2)
  ⇒ 20
[3]
Svc.run(int)  args=(3)
  ⇒ 30
`
	if got != want {
		t.Errorf("view differs\n--- got\n%s\n--- want\n%s", got, want)
	}
}

func TestRenderKeepsGroupingWithFullNames(t *testing.T) {
	line := `{"kind":"hit","probe":"p","point":"entry","method":"com.shop.Svc.run(Item)","call":N,"parent":null,"pid":1,"what":{"args":[{"@type":"com.a.Item","id":1}]}}`
	dump := strings.ReplaceAll(line, "N", "1") + "\n" + strings.ReplaceAll(line, "N", "2")
	got := Render([]byte(dump), []string{"p"})
	if !strings.Contains(got, "executions: 2, distinct: 1") {
		t.Errorf("identical executions were not folded:\n%s", got)
	}
}
