package compact

import (
	"strings"
	"unicode/utf8"
)

const maxWidth = 160

type piece struct {
	text string
	br   *bracket
}

type bracket struct {
	open  byte
	close byte
	items [][]piece
}

func wrapLine(line string, width int) []string {
	if utf8.RuneCountInString(line) <= width {
		return []string{line}
	}
	p := &parser{s: line}
	items, ok := p.seq(0, false)
	if !ok {
		return []string{line}
	}
	indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
	pr := &printer{width: width}
	pr.emit(items[0], indent)
	pr.flush()
	return pr.lines
}

type parser struct {
	s string
	i int
}

type seqState struct {
	items [][]piece
	cur   []piece
	start int
}

func (st *seqState) text(s string, end int) {
	if end > st.start {
		st.cur = append(st.cur, piece{text: s[st.start:end]})
	}
}

func (p *parser) seq(closing byte, split bool) ([][]piece, bool) {
	st := &seqState{start: p.i}
	for p.i < len(p.s) {
		c := p.s[p.i]
		switch {
		case c == '"':
			p.skipString()
		case isOpener(c):
			if !p.nested(st, c) {
				return nil, false
			}
			continue
		case closing != 0 && c == closing:
			st.text(p.s, p.i)
			p.i++
			return append(st.items, st.cur), true
		case isCloser(c):
			return nil, false
		case split && p.atSeparator():
			st.text(p.s, p.i)
			st.items = append(st.items, st.cur)
			st.cur = nil
			p.i += 2
			st.start = p.i
			continue
		}
		p.i++
	}
	if closing != 0 {
		return nil, false
	}
	st.text(p.s, p.i)
	return append(st.items, st.cur), true
}

func (p *parser) nested(st *seqState, open byte) bool {
	st.text(p.s, p.i)
	p.i++
	inner, ok := p.seq(closerOf(open), true)
	if !ok {
		return false
	}
	st.cur = append(st.cur, piece{br: &bracket{open: open, close: closerOf(open), items: inner}})
	st.start = p.i
	return true
}

func (p *parser) atSeparator() bool {
	return p.s[p.i] == ',' && p.i+1 < len(p.s) && p.s[p.i+1] == ' '
}

func isOpener(c byte) bool {
	return c == '{' || c == '[' || c == '('
}

func isCloser(c byte) bool {
	return c == '}' || c == ']' || c == ')'
}

func (p *parser) skipString() {
	for p.i++; p.i < len(p.s); p.i++ {
		switch p.s[p.i] {
		case '\\':
			p.i++
		case '"':
			return
		}
	}
}

func closerOf(open byte) byte {
	switch open {
	case '{':
		return '}'
	case '[':
		return ']'
	}
	return ')'
}

type printer struct {
	width int
	lines []string
	cur   string
}

func (p *printer) fits(extra string) bool {
	return utf8.RuneCountInString(p.cur)+utf8.RuneCountInString(extra) <= p.width
}

func (p *printer) flush() {
	p.lines = append(p.lines, p.cur)
	p.cur = ""
}

func (p *printer) emit(pieces []piece, indent string) {
	for _, pc := range pieces {
		if pc.br == nil {
			p.cur += pc.text
			continue
		}
		if f := flatBracket(pc.br); p.fits(f) {
			p.cur += f
			continue
		}
		p.cur += string(pc.br.open)
		p.flush()
		p.items(pc.br.items, indent+"  ")
		p.flush()
		p.cur = indent + string(pc.br.close)
	}
}

func (p *printer) items(items [][]piece, indent string) {
	p.cur = indent
	packed := false
	for i, it := range items {
		f := flat(it)
		if packed && p.fits(", "+f) {
			p.cur += ", " + f
			continue
		}
		if i > 0 {
			p.cur += ","
			p.flush()
			p.cur = indent
		}
		if p.fits(f) {
			p.cur += f
			packed = true
			continue
		}
		p.emit(it, indent)
		packed = false
	}
}

func flat(pieces []piece) string {
	var b strings.Builder
	for _, pc := range pieces {
		if pc.br == nil {
			b.WriteString(pc.text)
		} else {
			b.WriteString(flatBracket(pc.br))
		}
	}
	return b.String()
}

func flatBracket(g *bracket) string {
	parts := make([]string, len(g.items))
	for i, it := range g.items {
		parts[i] = flat(it)
	}
	return string(g.open) + strings.Join(parts, ", ") + string(g.close)
}
