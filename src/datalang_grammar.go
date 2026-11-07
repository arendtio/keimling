package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type gNode struct {
	kind string
	str  string
	re   *regexp.Regexp
	n, m int
	name string
	kids []gNode
}

type gRule struct {
	name string
	body gNode
	ret  *ex
	all  map[string]*gRule
}

type gOut struct {
	pos int
	ok  bool
	val any
	has bool
}

type gStat struct {
	rules    map[string]*gRule
	deadline dlEnv
	stack    []gFrame
	far      int
	exp      []string
}

type gFrame struct {
	name string
	pos  int
}

func (st *gStat) tick() error { return st.deadline.tick() }

type gRunner struct {
	env *dlEnv
}

func gParse(text, rule any, env *dlEnv) (any, error) {
	s, r, err := grammarArgs(text, rule)
	if err != nil {
		return nil, err
	}
	st := newGStat(r, env)
	out, err := st.matchRule(r, s, 0)
	if err != nil {
		return nil, err
	}
	if !out.ok {
		return nil, st.fail(s, "complete match")
	}
	if out.pos != len(s) {
		st.miss(out.pos, "end of input")
		return nil, st.fail(s, "end of input")
	}
	return out.val, nil
}

func gExtract(text, rule any, env *dlEnv) (any, error) {
	s, r, err := grammarArgs(text, rule)
	if err != nil {
		return nil, err
	}
	st := newGStat(r, env)
	out := []any{}
	for i := 0; i < len(s); {
		if err := st.tick(); err != nil {
			return nil, err
		}
		got, err := st.matchRule(r, s, i)
		if err != nil {
			return nil, err
		}
		if got.ok {
			if got.pos == i {
				line, col := locAt(s, i)
				return nil, &dataError{Code: "EMPTY_MATCH", Message: "extract match consumed no input", Offset: i, Line: line, Column: col}
			}
			out = append(out, got.val)
			i = got.pos
			continue
		}
		i++
	}
	return out, nil
}

func grammarArgs(text, rule any) (string, *gRule, error) {
	s, ok := text.(string)
	if !ok {
		return "", nil, dataErr("TYPE_ERROR", "parse expects text")
	}
	r, ok := rule.(*gRule)
	if !ok || r == nil {
		return "", nil, dataErr("TYPE_ERROR", "parse expects a grammar rule")
	}
	return s, r, nil
}

func newGStat(r *gRule, env *dlEnv) *gStat {
	var deadline time.Time
	if env != nil {
		deadline = env.deadline
	}
	return &gStat{rules: r.all, deadline: dlEnv{deadline: deadline}}
}

func (st *gStat) fail(input, what string) error {
	off := st.far
	line, col := locAt(input, off)
	exp := strings.Join(uniqueStrings(st.exp), ", ")
	if exp == "" {
		exp = what
	}
	return &dataError{Code: "PARSE_ERROR", Message: "parse failed", Offset: off, Line: line, Column: col, Expected: exp}
}

func (st *gStat) miss(pos int, what string) {
	if pos > st.far {
		st.far = pos
		st.exp = []string{what}
		return
	}
	if pos == st.far {
		st.exp = appendGrammarExpected(st.exp, what)
	}
}

func appendGrammarExpected(xs []string, s string) []string {
	for _, x := range xs {
		if x == s {
			return xs
		}
	}
	return append(xs, s)
}

func uniqueStrings(xs []string) []string { return xs }

func (st *gStat) matchRule(rule *gRule, in string, pos int) (gOut, error) {
	if rule == nil {
		return gOut{}, dataErr("PARSE_ERROR", "missing rule")
	}
	for _, f := range st.stack {
		if f.name == rule.name && f.pos == pos {
			return gOut{}, &dataError{Code: "LEFT_RECURSION", Message: "rule " + rule.name + " recurses without consuming input", Offset: pos}
		}
	}
	if len(st.stack) > 256 {
		return gOut{}, dataErr("LEFT_RECURSION", "grammar nested too deeply")
	}
	st.stack = append(st.stack, gFrame{rule.name, pos})
	defer func() { st.stack = st.stack[:len(st.stack)-1] }()
	caps := map[string]any{}
	out, err := st.match(rule.body, in, pos, caps)
	if err != nil || !out.ok {
		return out, err
	}
	if rule.ret != nil {
		env := &dlEnv{vars: map[string]any{}, deadline: st.deadline.deadline}
		for k, v := range caps {
			env.vars[k] = v
		}
		val, err := eval(rule.ret, env)
		if err != nil {
			return gOut{}, err
		}
		out.val, out.has = val, true
		return out, nil
	}
	if len(caps) > 0 {
		obj := map[string]any{}
		for k, v := range caps {
			obj[k] = v
		}
		out.val, out.has = obj, true
		return out, nil
	}
	out.val, out.has = in[pos:out.pos], true
	return out, nil
}

func (st *gStat) match(n gNode, in string, pos int, caps map[string]any) (gOut, error) {
	if err := st.tick(); err != nil {
		return gOut{}, err
	}
	switch n.kind {
	case "empty":
		return gOut{pos: pos, ok: true}, nil
	case "lit":
		if strings.HasPrefix(in[pos:], n.str) {
			return gOut{pos: pos + len(n.str), ok: true}, nil
		}
		st.miss(pos, strconv.Quote(n.str))
		return gOut{pos: pos}, nil
	case "re":
		loc := n.re.FindStringIndex(in[pos:])
		if loc == nil || loc[0] != 0 {
			st.miss(pos, n.str)
			return gOut{pos: pos}, nil
		}
		end := pos + loc[1]
		return gOut{pos: end, ok: true, val: in[pos:end], has: true}, nil
	case "char":
		if pos >= len(in) {
			st.miss(pos, fmt.Sprintf("char(%d)", n.n))
			return gOut{pos: pos}, nil
		}
		r, size := utf8.DecodeRuneInString(in[pos:])
		if r != rune(n.n) {
			st.miss(pos, fmt.Sprintf("char(%d)", n.n))
			return gOut{pos: pos}, nil
		}
		return gOut{pos: pos + size, ok: true, val: string(r), has: true}, nil
	case "prim":
		return st.matchPrim(n.str, in, pos)
	case "call":
		rule := st.rules[n.str]
		if rule == nil {
			return gOut{}, &dataError{Code: "PARSE_ERROR", Message: "unknown rule " + n.str, Offset: pos}
		}
		return st.matchRule(rule, in, pos)
	case "seq":
		cur := pos
		for _, k := range n.kids {
			out, err := st.match(k, in, cur, caps)
			if err != nil {
				return gOut{}, err
			}
			if !out.ok {
				return gOut{pos: pos}, nil
			}
			cur = out.pos
		}
		return gOut{pos: cur, ok: true}, nil
	case "alt":
		for _, k := range n.kids {
			iter := cloneCaps(caps)
			out, err := st.match(k, in, pos, iter)
			if err != nil {
				return gOut{}, err
			}
			if out.ok {
				replaceCaps(caps, iter)
				return out, nil
			}
		}
		return gOut{pos: pos}, nil
	case "opt":
		out, err := st.match(n.kids[0], in, pos, caps)
		if err != nil {
			return gOut{}, err
		}
		if !out.ok {
			return gOut{pos: pos, ok: true, val: nil, has: true}, nil
		}
		return out, nil
	case "star", "plus":
		return st.matchRepeat(n, in, pos, caps)
	case "rep":
		return st.matchBounded(n, in, pos, caps)
	case "cap":
		start := pos
		out, err := st.match(n.kids[0], in, pos, caps)
		if err != nil {
			return gOut{}, err
		}
		if !out.ok {
			return gOut{pos: pos}, nil
		}
		if _, exists := caps[n.name]; exists {
			return gOut{}, &dataError{Code: "DUPLICATE_CAPTURE", Message: "capture " + n.name + " is set twice", Offset: start}
		}
		var val any
		if out.has {
			val = out.val
		} else {
			val = in[start:out.pos]
		}
		caps[n.name] = val
		return gOut{pos: out.pos, ok: true, val: val, has: true}, nil
	case "back":
		v, ok := caps[n.name]
		s, isStr := v.(string)
		if !ok || !isStr {
			st.miss(pos, "="+n.name)
			return gOut{pos: pos}, nil
		}
		if strings.HasPrefix(in[pos:], s) {
			return gOut{pos: pos + len(s), ok: true}, nil
		}
		st.miss(pos, "="+n.name)
		return gOut{pos: pos}, nil
	case "text":
		start := pos
		iter := cloneCaps(caps)
		out, err := st.match(n.kids[0], in, pos, iter)
		if err != nil || !out.ok {
			return out, err
		}
		return gOut{pos: out.pos, ok: true, val: in[start:out.pos], has: true}, nil
	case "sep":
		return st.matchSep(n, in, pos, caps)
	case "until":
		idx := strings.Index(in[pos:], n.str)
		if idx < 0 {
			st.miss(pos, "until "+strconv.Quote(n.str))
			return gOut{pos: pos}, nil
		}
		return gOut{pos: pos + idx, ok: true, val: in[pos : pos+idx], has: true}, nil
	case "and":
		iter := cloneCaps(caps)
		out, err := st.match(n.kids[0], in, pos, iter)
		if err != nil {
			return gOut{}, err
		}
		if !out.ok {
			return gOut{pos: pos}, nil
		}
		return gOut{pos: pos, ok: true}, nil
	case "not":
		iter := cloneCaps(caps)
		out, err := st.match(n.kids[0], in, pos, iter)
		if err != nil {
			return gOut{}, err
		}
		if out.ok {
			st.miss(pos, "not "+n.kids[0].kind)
			return gOut{pos: pos}, nil
		}
		return gOut{pos: pos, ok: true}, nil
	default:
		return gOut{}, dataErr("PARSE_ERROR", "unknown grammar node "+n.kind)
	}
}

func (st *gStat) matchRepeat(n gNode, in string, pos int, caps map[string]any) (gOut, error) {
	start := pos
	child := n.kids[0]
	collect := nodeProduces(child)
	var items []any
	count := 0
	for {
		if err := st.tick(); err != nil {
			return gOut{}, err
		}
		iterStart := pos
		iter := cloneCaps(caps)
		out, err := st.match(child, in, pos, iter)
		if err != nil {
			return gOut{}, err
		}
		if !out.ok || out.pos == pos {
			break
		}
		pos = out.pos
		count++
		if collect {
			if out.has {
				items = append(items, out.val)
			} else {
				items = append(items, in[iterStart:pos])
			}
		}
	}
	if n.kind == "plus" && count == 0 {
		return gOut{pos: start}, nil
	}
	if collect {
		if items == nil {
			items = []any{}
		}
		return gOut{pos: pos, ok: true, val: items, has: true}, nil
	}
	return gOut{pos: pos, ok: true}, nil
}

func (st *gStat) matchBounded(n gNode, in string, pos int, caps map[string]any) (gOut, error) {
	child := n.kids[0]
	collect := nodeProduces(child)
	var items []any
	count := 0
	for count < n.m {
		iter := cloneCaps(caps)
		out, err := st.match(child, in, pos, iter)
		if err != nil {
			return gOut{}, err
		}
		if !out.ok || out.pos == pos {
			break
		}
		pos = out.pos
		count++
		if collect {
			if out.has {
				items = append(items, out.val)
			}
		}
	}
	if count < n.n {
		return gOut{pos: pos}, nil
	}
	if collect {
		if items == nil {
			items = []any{}
		}
		return gOut{pos: pos, ok: true, val: items, has: true}, nil
	}
	return gOut{pos: pos, ok: true}, nil
}

func (st *gStat) matchSep(n gNode, in string, pos int, caps map[string]any) (gOut, error) {
	first, err := st.match(n.kids[0], in, pos, caps)
	if err != nil || !first.ok {
		return first, err
	}
	items := []any{valueOfMatch(first, in, pos)}
	pos = first.pos
	for {
		save := pos
		sep, err := st.match(n.kids[1], in, pos, caps)
		if err != nil {
			return gOut{}, err
		}
		if !sep.ok {
			break
		}
		item, err := st.match(n.kids[0], in, sep.pos, caps)
		if err != nil {
			return gOut{}, err
		}
		if !item.ok {
			pos = save
			break
		}
		items = append(items, valueOfMatch(item, in, sep.pos))
		pos = item.pos
	}
	return gOut{pos: pos, ok: true, val: items, has: true}, nil
}

func valueOfMatch(out gOut, in string, start int) any {
	if out.has {
		return out.val
	}
	return in[start:out.pos]
}

func (st *gStat) matchPrim(kind, in string, pos int) (gOut, error) {
	switch kind {
	case "int":
		i := pos
		if i < len(in) && (in[i] == '+' || in[i] == '-') {
			i++
		}
		start := i
		for i < len(in) && in[i] >= '0' && in[i] <= '9' {
			i++
		}
		if i == start {
			st.miss(pos, "int")
			return gOut{pos: pos}, nil
		}
		n, err := strconv.ParseInt(in[pos:i], 10, 64)
		if err != nil {
			return gOut{}, dataErr("PARSE_ERROR", "integer overflow")
		}
		return gOut{pos: i, ok: true, val: n, has: true}, nil
	case "decimal":
		i := pos
		if i < len(in) && (in[i] == '+' || in[i] == '-') {
			i++
		}
		start := i
		for i < len(in) && in[i] >= '0' && in[i] <= '9' {
			i++
		}
		if i == start || i >= len(in) || in[i] != '.' {
			st.miss(pos, "decimal")
			return gOut{pos: pos}, nil
		}
		i++
		frac := i
		for i < len(in) && in[i] >= '0' && in[i] <= '9' {
			i++
		}
		if i == frac {
			st.miss(pos, "decimal")
			return gOut{pos: pos}, nil
		}
		f, err := strconv.ParseFloat(in[pos:i], 64)
		if err != nil {
			return gOut{}, dataErr("PARSE_ERROR", "bad decimal")
		}
		return gOut{pos: i, ok: true, val: f, has: true}, nil
	case "word":
		i := pos
		for i < len(in) && ((in[i] >= 'A' && in[i] <= 'Z') || (in[i] >= 'a' && in[i] <= 'z')) {
			i++
		}
		if i == pos {
			st.miss(pos, "word")
			return gOut{pos: pos}, nil
		}
		return gOut{pos: i, ok: true, val: in[pos:i], has: true}, nil
	default:
		return gOut{}, dataErr("PARSE_ERROR", "unknown primitive "+kind)
	}
}

func nodeProduces(n gNode) bool {
	switch n.kind {
	case "call", "prim", "cap", "text", "sep", "until", "char", "re":
		return true
	case "opt", "star", "plus", "rep":
		return nodeProduces(n.kids[0])
	case "seq", "alt":
		for _, k := range n.kids {
			if k.kind == "cap" || nodeProduces(k) && k.kind != "lit" {
				return true
			}
		}
	}
	return false
}

func cloneCaps(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func replaceCaps(dst, src map[string]any) {
	for k := range dst {
		delete(dst, k)
	}
	for k, v := range src {
		dst[k] = v
	}
}

func (p *parser) parseGrammarBlock() (map[string]*gRule, error) {
	if !p.consumeOp("{") {
		return nil, p.err("expected {")
	}
	rules := map[string]*gRule{}
	var order []*gRule
	for {
		p.skip()
		if p.consumeOp("}") {
			break
		}
		if p.i >= len(p.s) {
			return nil, p.err("unterminated grammar")
		}
		name, err := p.ident()
		if err != nil {
			return nil, err
		}
		if !p.consumeOp("=") {
			return nil, p.err("expected =")
		}
		body, err := p.parseGAlt()
		if err != nil {
			return nil, err
		}
		rule := &gRule{name: name, body: body}
		if p.consumeOp("=>") {
			ret, err := p.parsePipe()
			if err != nil {
				return nil, err
			}
			rule.ret = ret
		}
		rules[name] = rule
		order = append(order, rule)
	}
	for _, rule := range order {
		rule.all = rules
	}
	return rules, nil
}

func (p *parser) parseGAlt() (gNode, error) {
	left, err := p.parseGSeq()
	if err != nil {
		return gNode{}, err
	}
	alts := []gNode{left}
	for p.consumeOp("|") {
		right, err := p.parseGSeq()
		if err != nil {
			return gNode{}, err
		}
		alts = append(alts, right)
	}
	if len(alts) == 1 {
		return left, nil
	}
	return gNode{kind: "alt", kids: alts}, nil
}

func (p *parser) parseGSeq() (gNode, error) {
	var kids []gNode
	for !p.gSeqStop() {
		n, err := p.parseGPrefix()
		if err != nil {
			return gNode{}, err
		}
		kids = append(kids, n)
	}
	if len(kids) == 0 {
		return gNode{}, p.err("expected grammar expression")
	}
	if len(kids) == 1 {
		return kids[0], nil
	}
	return gNode{kind: "seq", kids: kids}, nil
}

func (p *parser) gSeqStop() bool {
	p.skip()
	if p.i >= len(p.s) {
		return true
	}
	if p.starts("=>") || p.starts(")") || p.starts("}") || p.starts("|") {
		return true
	}
	if p.gdepth == 0 && p.lookingAtRule() {
		return true
	}
	return false
}

func (p *parser) lookingAtRule() bool {
	save := p.i
	id, ok := p.peekIdent()
	if !ok {
		p.i = save
		return false
	}
	p.i += len(id)
	p.skip()
	yes := p.i < len(p.s) && p.s[p.i] == '=' && !(p.i+1 < len(p.s) && p.s[p.i+1] == '>')
	p.i = save
	return yes
}

func (p *parser) parseGPrefix() (gNode, error) {
	if p.consumeOp("&") {
		inner, err := p.parseGPrefix()
		if err != nil {
			return gNode{}, err
		}
		return gNode{kind: "and", kids: []gNode{inner}}, nil
	}
	if p.consumeOp("!") {
		inner, err := p.parseGPrefix()
		if err != nil {
			return gNode{}, err
		}
		return gNode{kind: "not", kids: []gNode{inner}}, nil
	}
	if p.lookingAtCapture() {
		name, _ := p.ident()
		p.consumeOp(":")
		inner, err := p.parseGSuffix()
		if err != nil {
			return gNode{}, err
		}
		return gNode{kind: "cap", name: name, kids: []gNode{inner}}, nil
	}
	return p.parseGSuffix()
}

func (p *parser) lookingAtCapture() bool {
	save := p.i
	id, ok := p.peekIdent()
	if !ok {
		return false
	}
	p.i += len(id)
	p.skip()
	yes := p.i < len(p.s) && p.s[p.i] == ':'
	p.i = save
	return yes
}

func (p *parser) parseGSuffix() (gNode, error) {
	n, err := p.parseGPrimary()
	if err != nil {
		return gNode{}, err
	}
	for {
		switch {
		case p.consumeOp("?"):
			n = gNode{kind: "opt", kids: []gNode{n}}
		case p.consumeOp("*"):
			n = gNode{kind: "star", kids: []gNode{n}}
		case p.consumeOp("+"):
			n = gNode{kind: "plus", kids: []gNode{n}}
		case p.lookingAtRep():
			rep, err := p.parseRep(n)
			if err != nil {
				return gNode{}, err
			}
			n = rep
		default:
			return n, nil
		}
	}
}

func (p *parser) lookingAtRep() bool {
	p.skip()
	if p.i >= len(p.s) || p.s[p.i] != '{' {
		return false
	}
	j := p.i + 1
	for j < len(p.s) && (p.s[j] == ' ' || p.s[j] == '\t') {
		j++
	}
	return j < len(p.s) && p.s[j] >= '0' && p.s[j] <= '9'
}

func (p *parser) parseRep(inner gNode) (gNode, error) {
	if !p.consumeOp("{") {
		return gNode{}, p.err("expected {")
	}
	min, err := p.rawInt()
	if err != nil {
		return gNode{}, err
	}
	max := min
	if p.consumeOp(",") {
		p.skip()
		if p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
			max, err = p.rawInt()
			if err != nil {
				return gNode{}, err
			}
		} else {
			max = 1 << 20
		}
	}
	if !p.consumeOp("}") {
		return gNode{}, p.err("expected }")
	}
	return gNode{kind: "rep", n: min, m: max, kids: []gNode{inner}}, nil
}

func (p *parser) rawInt() (int, error) {
	p.skip()
	start := p.i
	for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
		p.i++
	}
	if start == p.i {
		return 0, p.err("expected integer")
	}
	n, err := strconv.Atoi(p.s[start:p.i])
	if err != nil {
		return 0, p.err("bad integer")
	}
	return n, nil
}

func (p *parser) parseGPrimary() (gNode, error) {
	p.skip()
	if p.i >= len(p.s) {
		return gNode{}, p.err("expected grammar expression")
	}
	if p.starts(`"""`) || p.followedByPattern() {
		ex, err := p.parsePatternExpr()
		if err != nil {
			return gNode{}, err
		}
		return ex.rule.body, nil
	}
	switch p.s[p.i] {
	case '"':
		s, err := p.parseString()
		if err != nil {
			return gNode{}, err
		}
		return gNode{kind: "lit", str: s}, nil
	case '/':
		return p.parseRegex()
	case '(':
		p.i++
		p.gdepth++
		n, err := p.parseGAlt()
		p.gdepth--
		if err != nil {
			return gNode{}, err
		}
		if !p.consumeOp(")") {
			return gNode{}, p.err("expected )")
		}
		return n, nil
	}
	if p.consumeOp("=") {
		name, ok := p.peekIdent()
		if !ok {
			return gNode{}, p.err("expected capture name after =")
		}
		p.i += len(name)
		return gNode{kind: "back", name: name}, nil
	}
	id, ok := p.peekIdent()
	if !ok {
		return gNode{}, p.err("expected grammar expression")
	}
	p.i += len(id)
	p.skip()
	if p.i < len(p.s) && p.s[p.i] == '(' && grammarCall(id) {
		p.i++
		switch id {
		case "char":
			n, err := p.rawInt()
			if err != nil {
				return gNode{}, err
			}
			if !p.consumeOp(")") {
				return gNode{}, p.err("expected )")
			}
			return gNode{kind: "char", n: n}, nil
		case "until":
			s, err := p.parseString()
			if err != nil {
				return gNode{}, err
			}
			if !p.consumeOp(")") {
				return gNode{}, p.err("expected )")
			}
			return gNode{kind: "until", str: s}, nil
		case "sepBy":
			item, err := p.parseGAlt()
			if err != nil {
				return gNode{}, err
			}
			if !p.consumeOp(",") {
				return gNode{}, p.err("expected ,")
			}
			sep, err := p.parseGAlt()
			if err != nil {
				return gNode{}, err
			}
			if !p.consumeOp(")") {
				return gNode{}, p.err("expected )")
			}
			return gNode{kind: "sep", kids: []gNode{item, sep}}, nil
		case "text":
			inner, err := p.parseGAlt()
			if err != nil {
				return gNode{}, err
			}
			if !p.consumeOp(")") {
				return gNode{}, p.err("expected )")
			}
			return gNode{kind: "text", kids: []gNode{inner}}, nil
		default:
			return gNode{}, p.err("unknown grammar function " + id)
		}
	}
	switch id {
	case "int", "decimal", "word":
		return gNode{kind: "prim", str: id}, nil
	default:
		return gNode{kind: "call", str: id}, nil
	}
}

func regexRule(n gNode) *gRule {
	r := &gRule{name: "pattern", body: n}
	r.all = map[string]*gRule{r.name: r}
	return r
}

func grammarCall(id string) bool {
	switch id {
	case "char", "until", "sepBy", "text":
		return true
	default:
		return false
	}
}

func (p *parser) parseRegex() (gNode, error) {
	if p.i >= len(p.s) || p.s[p.i] != '/' {
		return gNode{}, p.err("expected regex")
	}
	p.i++
	start := p.i
	var b strings.Builder
	for p.i < len(p.s) {
		if p.s[p.i] == '\\' && p.i+1 < len(p.s) {
			b.WriteByte(p.s[p.i])
			b.WriteByte(p.s[p.i+1])
			p.i += 2
			continue
		}
		if p.s[p.i] == '/' {
			re, err := regexp.Compile(b.String())
			if err != nil {
				return gNode{}, &dataError{Code: "SYNTAX", Message: err.Error(), Offset: start}
			}
			p.i++
			return gNode{kind: "re", str: b.String(), re: re}, nil
		}
		b.WriteByte(p.s[p.i])
		p.i++
	}
	return gNode{}, p.err("unterminated regex")
}

func compilePattern(s string) (*gRule, error) {
	body, err := patternBody(s)
	if err != nil {
		return nil, err
	}
	rule := &gRule{name: "pattern", body: body, all: map[string]*gRule{}}
	rule.all["pattern"] = rule
	return rule, nil
}

func patternBody(s string) (gNode, error) {
	var kids []gNode
	var lit strings.Builder
	flush := func() {
		if lit.Len() == 0 {
			return
		}
		kids = append(kids, gNode{kind: "lit", str: lit.String()})
		lit.Reset()
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' {
			esc, n, err := decodeEscape(s[i:])
			if err != nil {
				return gNode{}, dataErr("SYNTAX", err.Error())
			}
			lit.WriteString(esc)
			i += n - 1
			continue
		}
		if strings.HasPrefix(s[i:], "{{") {
			lit.WriteByte('{')
			i++
			continue
		}
		if strings.HasPrefix(s[i:], "}}") {
			lit.WriteByte('}')
			i++
			continue
		}
		if s[i] == '{' {
			flush()
			end, err := matchingBrace(s, i)
			if err != nil {
				return gNode{}, err
			}
			inner := s[i+1 : end]
			p := &parser{s: inner}
			node, err := p.parseGPrefix()
			if err != nil {
				return gNode{}, err
			}
			p.skip()
			if p.i != len(p.s) {
				return gNode{}, p.err("trailing pattern capture")
			}
			kids = append(kids, node)
			i = end
			continue
		}
		lit.WriteByte(s[i])
	}
	flush()
	if len(kids) == 0 {
		return gNode{kind: "empty"}, nil
	}
	if len(kids) == 1 {
		return kids[0], nil
	}
	return gNode{kind: "seq", kids: kids}, nil
}

func matchingBrace(s string, open int) (int, error) {
	depth := 0
	for i := open; i < len(s); i++ {
		if s[i] == '"' {
			i++
			for i < len(s) && s[i] != '"' {
				if s[i] == '\\' {
					i++
				}
				i++
			}
			continue
		}
		if s[i] == '{' {
			depth++
		}
		if s[i] == '}' {
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, dataErr("SYNTAX", "unterminated pattern capture")
}
