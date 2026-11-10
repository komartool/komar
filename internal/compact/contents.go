package compact

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	headWidth    = 100
	outcomeWidth = 40
)

type layout struct {
	contentsLines   int
	contentsBytes   int
	contentsEntries int
}

var defaultLayout = layout{contentsLines: 2000, contentsBytes: 50_000, contentsEntries: 60}

type block struct {
	head    string
	outcome string
	count   int
	lines   []string
}

func newBlock(body []string, count int) block {
	b := block{head: body[0], count: count}
	for _, line := range body {
		if strings.HasPrefix(line, "  ⇒ ") {
			b.outcome = strings.TrimPrefix(line, "  ⇒ ")
		}
	}
	if count > 1 {
		body[0] += "   ×" + strconv.Itoa(count)
	}
	for _, line := range body {
		b.lines = append(b.lines, wrapLine(line, maxWidth)...)
	}
	return b
}

func (lay layout) body(header []string, blocks []block) []string {
	plain := 0
	size := 0
	for _, s := range header {
		size += len(s) + 1
	}
	for _, b := range blocks {
		plain += len(b.lines) + 1
		for _, s := range b.lines {
			size += len(s) + 1
		}
	}
	if len(header)+plain <= lay.contentsLines && size <= lay.contentsBytes {
		return joinBlocks(blocks, func(int) string { return "" })
	}
	return lay.withContents(len(header), blocks)
}

func (lay layout) withContents(headerLines int, blocks []block) []string {
	listed := min(len(blocks), lay.contentsEntries)
	tocLines := 1 + listed
	if listed < len(blocks) {
		tocLines++
	}
	starts := make([]int, len(blocks))
	next := headerLines + tocLines + 1
	for i, b := range blocks {
		starts[i] = next
		next += len(b.lines) + 1
	}
	out := []string{"contents (line numbers in this file):"}
	for i := 0; i < listed; i++ {
		out = append(out, "  "+marker(i)+" L"+strconv.Itoa(starts[i])+" "+blocks[i].entry())
	}
	if listed < len(blocks) {
		out = append(out, "  … "+strconv.Itoa(len(blocks)-listed)+" more, lines "+
			strconv.Itoa(starts[listed])+"–"+strconv.Itoa(next-1))
	}
	return append(out, joinBlocks(blocks, marker)...)
}

func (b block) entry() string {
	s := ""
	if b.count > 1 {
		s = "×" + strconv.Itoa(b.count) + " "
	}
	s += clip(withoutThis(b.head), headWidth)
	if b.outcome != "" {
		result, _, _ := strings.Cut(b.outcome, "  ")
		s += " ⇒ " + clip(result, outcomeWidth)
	}
	return s
}

func withoutThis(head string) string {
	parts := strings.Split(head, "  ")
	kept := parts[:1]
	for _, p := range parts[1:] {
		if !strings.HasPrefix(p, "this=") {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, "  ")
}

func joinBlocks(blocks []block, separator func(int) string) []string {
	var out []string
	for i, b := range blocks {
		out = append(out, separator(i))
		out = append(out, b.lines...)
	}
	return out
}

func marker(i int) string {
	return "[" + strconv.Itoa(i+1) + "]"
}

func clip(s string, width int) string {
	if utf8.RuneCountInString(s) <= width {
		return s
	}
	r := []rune(s)
	return string(r[:width-1]) + "…"
}
