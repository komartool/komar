package compact

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestWrapLine(t *testing.T) {
	cases := []struct {
		name  string
		line  string
		width int
		want  []string
	}{
		{
			name:  "a line that fits stays as it is",
			line:  `Svc.run(Order)  args=(Order#1{id: 1})`,
			width: 80,
			want:  []string{`Svc.run(Order)  args=(Order#1{id: 1})`},
		},
		{
			name:  "a long value breaks at its brackets and packs fields that fit",
			line:  `Svc.run(Order)  args=(Order#1{id: 1, customer: "ann", tags: ["a", "b", "c"]})`,
			width: 30,
			want: []string{
				`Svc.run(Order)  args=(`,
				`  Order#1{`,
				`    id: 1, customer: "ann",`,
				`    tags: ["a", "b", "c"]`,
				`  }`,
				`)`,
			},
		},
		{
			name:  "continuation lines are indented from the line's own indent",
			line:  `    → Repo.save(Item{key: "k1", value: "v1"}) ⇒ ok`,
			width: 30,
			want: []string{
				`    → Repo.save(`,
				`      Item{`,
				`        key: "k1", value: "v1"`,
				`      }`,
				`    ) ⇒ ok`,
			},
		},
		{
			name:  "commas and brackets inside strings are not split",
			line:  `call("a, [b", "cc")`,
			width: 10,
			want:  []string{`call(`, `  "a, [b",`, `  "cc"`, `)`},
		},
		{
			name:  "unbalanced text is left alone",
			line:  `<error: no field ')' in X> and a long tail of words`,
			width: 10,
			want:  []string{`<error: no field ')' in X> and a long tail of words`},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := wrapLine(c.line, c.width)
			if strings.Join(got, "\n") != strings.Join(c.want, "\n") {
				t.Errorf("wrap differs\n--- got\n%s\n--- want\n%s", strings.Join(got, "\n"), strings.Join(c.want, "\n"))
			}
		})
	}
}

func TestRenderWrapsLongLines(t *testing.T) {
	var fields []string
	for _, k := range []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta", "iota", "kappa"} {
		fields = append(fields, `"`+k+`":"value of `+k+` that is long enough"`)
	}
	dump := `{"kind":"hit","probe":"p","point":"entry","method":"Svc.run(Order)","call":1,"parent":null,"pid":1,` +
		`"what":{"args":[{"@type":"com.shop.Order",` + strings.Join(fields, ",") + `}]}}`
	got := Render([]byte(dump), []string{"p"})
	body := strings.SplitN(got, "executions:", 2)[1]
	for _, line := range strings.Split(body, "\n") {
		if utf8.RuneCountInString(line) > maxWidth {
			t.Errorf("line longer than %d: %s", maxWidth, line)
		}
	}
	if !strings.Contains(body, "  com.shop.Order#1{\n") {
		t.Errorf("object was not broken onto its own lines:\n%s", body)
	}
}
