package compact

import (
	"regexp"
	"strconv"
	"strings"
)

var proxyMarker = regexp.MustCompile(`\$\$EnhancerBy|\$\$SpringCGLIB|\$HibernateProxy|\$MockitoMock\$|\$ByteBuddy\$|^\$|\$Intercepted$`)

type names struct {
	seen  map[string]bool
	owner map[string]string
}

func newNames() *names {
	return &names{seen: map[string]bool{}, owner: map[string]string{}}
}

func (n *names) typ(full string) string {
	if full == "" {
		return ""
	}
	if n == nil {
		return full
	}
	short := shortType(full)
	if n.seen[full] && n.owner[short] == full {
		return short
	}
	n.seen[full] = true
	if _, taken := n.owner[short]; !taken {
		n.owner[short] = full
	}
	return full
}

func (n *names) method(m string) string {
	name, args, _ := strings.Cut(m, "(")
	dot := strings.LastIndexByte(name, '.')
	if dot < 0 {
		return m
	}
	return n.typ(name[:dot]) + name[dot:] + "(" + args
}

func (n *names) callName(m string) string {
	name, _, _ := strings.Cut(n.method(m), "(")
	return name
}

func proxyOf(full string) (string, bool) {
	pkg, simple := "", full
	if dot := strings.LastIndexByte(full, '.'); dot >= 0 {
		pkg, simple = full[:dot+1], full[dot+1:]
	}
	loc := proxyMarker.FindStringIndex(simple)
	if loc == nil {
		return "", false
	}
	if loc[0] == 0 {
		if strings.HasPrefix(simple, "$Proxy") {
			return full, true
		}
		simple = strings.TrimPrefix(simple, "$")
		if cut := strings.Index(simple, "$"); cut >= 0 {
			return pkg + simple[:cut], true
		}
		return pkg + simple, true
	}
	return pkg + simple[:loc[0]], true
}

func shortType(t string) string {
	t = t[strings.LastIndexByte(t, '.')+1:]
	i := strings.LastIndexByte(t, '$')
	if i < 0 {
		return t
	}
	if _, err := strconv.Atoi(t[i+1:]); err == nil {
		return t
	}
	return t[i+1:]
}
