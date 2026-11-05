package jvm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"komar/internal/spec"
	"os"
	"path/filepath"
	"strings"

	_ "embed"
)

//go:embed komar-agent.jar
var agentJar []byte

const (
	jarOverrideEnv = "KOMAR_AGENT_JAR"
	toolOptionsEnv = "JAVA_TOOL_OPTIONS"
	shortHashLen   = 16
)

type Session struct {
	dir  string
	file string
	jar  string
}

type sessionFile struct {
	Host   string         `json:"host"`
	Port   int            `json:"port"`
	Probes []sessionProbe `json:"probes"`
}

type sessionProbe struct {
	ID       string        `json:"id"`
	Class    string        `json:"class"`
	Subtypes bool          `json:"subtypes"`
	Method   string        `json:"method"`
	Params   []string      `json:"params"`
	Lines    []int         `json:"lines"`
	Call     *sessionCall  `json:"call"`
	At       []string      `json:"at"`
	When     *sessionItem  `json:"when"`
	What     []sessionItem `json:"what"`
	Limits   spec.Limits   `json:"limits"`
}

type sessionCall struct {
	Class    string   `json:"class"`
	Subtypes bool     `json:"subtypes"`
	Method   string   `json:"method"`
	Params   []string `json:"params"`
}

type sessionItem struct {
	Key        string   `json:"key"`
	Expr       string   `json:"expr"`
	Roots      []string `json:"roots"`
	Vars       []string `json:"vars"`
	CallFields []string `json:"call_fields"`
}

func item(key string, e spec.Expr) sessionItem {
	return sessionItem{
		Key: key, Expr: e.Text, Roots: nonNil(e.Roots), Vars: nonNil(e.Vars), CallFields: nonNil(e.CallFields),
	}
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func encodeSession(host string, port int, s *spec.Spec) ([]byte, error) {
	f := sessionFile{Host: host, Port: port, Probes: make([]sessionProbe, 0, len(s.Probes))}
	for _, p := range s.Probes {
		sp := sessionProbe{
			ID: p.ID, Class: p.Where.Class, Subtypes: p.Where.Subtypes, Method: p.Where.Method,
			At: p.Where.At, Lines: p.Where.Lines, Limits: p.Limits, What: make([]sessionItem, 0, len(p.What)),
		}
		if p.Where.HasParam {
			sp.Params = append([]string{}, p.Where.Params...)
		}
		if c := p.Where.Call; c != nil {
			sp.Call = &sessionCall{Class: c.Class, Subtypes: c.Subtypes, Method: c.Method}
			if c.HasParam {
				sp.Call.Params = append([]string{}, c.Params...)
			}
		}
		if p.When != nil {
			w := item("when", *p.When)
			sp.When = &w
		}
		for _, it := range p.What {
			sp.What = append(sp.What, item(it.Key, it.Expr))
		}
		f.Probes = append(f.Probes, sp)
	}
	data, err := json.Marshal(f)
	if err != nil {
		return nil, fmt.Errorf("encode session: %w", err)
	}
	return data, nil
}

func Prepare(host string, port int, s *spec.Spec) (*Session, error) {
	content, err := encodeSession(host, port, s)
	if err != nil {
		return nil, err
	}
	jar, err := agentPath()
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "komar-")
	if err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}
	file := filepath.Join(dir, "session.json")
	if err := os.WriteFile(file, content, 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("write session file: %w", err)
	}
	return &Session{dir: dir, file: file, jar: jar}, nil
}

func (s *Session) Env(base []string) []string {
	option := fmt.Sprintf("-javaagent:%s=%s", s.jar, s.file)
	env := make([]string, 0, len(base)+1)
	existing := ""
	for _, kv := range base {
		if value, ok := strings.CutPrefix(kv, toolOptionsEnv+"="); ok {
			existing = value
			continue
		}
		env = append(env, kv)
	}
	return append(env, toolOptionsEnv+"="+strings.TrimSpace(existing+" "+option))
}

func (s *Session) Close() error {
	if err := os.RemoveAll(s.dir); err != nil {
		return fmt.Errorf("remove session dir: %w", err)
	}
	return nil
}

func agentPath() (string, error) {
	if path := os.Getenv(jarOverrideEnv); path != "" {
		return path, nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find cache dir: %w", err)
	}
	sum := sha256.Sum256(agentJar)
	path := filepath.Join(cache, "komar", "agent-"+hex.EncodeToString(sum[:])[:shortHashLen]+".jar")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	return path, extract(path)
}

func extract(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "agent-*.tmp")
	if err != nil {
		return fmt.Errorf("extract agent: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(agentJar); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("extract agent: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("extract agent: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("extract agent: %w", err)
	}
	return nil
}
