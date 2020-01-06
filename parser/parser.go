package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

//go:generate stringer -type ItemStatus
//go:generate stringer -type ItemType
type Todofile struct {
	Items []*Todoitem
}

type Node interface {
	Print(children bool, w io.Writer)
	Children() []*Todoitem
}

type (
	ItemType   int
	ItemStatus int
	TagType    int
)

const (
	ItText ItemType = iota
	ItItem
	ItTitle
)

const (
	StUnknown ItemStatus = iota
	StPending
	StStarted
	StDone
	StCancel
	StArchive
)

const (
	TagNormal TagType = iota
	TagTime
	TagDone
	TagStarted
	TagLasted
	TagEst
	TagCritical
	TagHigh
	TagLow
	TagToday
	TagBold
	TagItalic
	TagDeleted
	TagCode

	TagUnknown
)

type Tag struct {
	Start, Stop int
	Type        TagType
	Text        string
}

type Todoitem struct {
	Ident       int
	Type        ItemType
	Token       string
	Text        string
	Status      ItemStatus
	StatusTimes map[ItemStatus]time.Time
	Tags        []Tag
	Assignees   []Tag
	Items       []*Todoitem
	parent      *Todoitem

	offset int
}

func (item ItemType) MarshalJSON() ([]byte, error) {
	return json.Marshal(item.String())
}

func (state ItemStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(state.String())
}

func Parse(filename string, r io.Reader) (*Todofile, error) {
	var (
		todofile           Todofile
		s                  = bufio.NewScanner(r)
		prev, parent, node *Todoitem
		err                error
		// prev     = root
		// stack    = make([]*Todoitem, 0)
	)

	s.Split(bufio.ScanLines)
	for s.Scan() {
		line := s.Text()
		if len(strings.TrimSpace(line)) == 0 {
			continue // skip
		}

		parent, node, err = parseNode(prev, parent, line, &todofile)
		if err != nil {
			return nil, err
		}
		prev = node
	}

	return &todofile, nil
}

func (node *Todoitem) Add(child *Todoitem) {
	child.parent = node
	node.Items = append(node.Items, child)
}

func (node *Todoitem) String() string {
	return strings.Repeat(" ", node.Ident) + node.Text
}

func (node *Todoitem) Print(children bool, w io.Writer) {
	fmt.Fprintf(w, "%s\n", node.String())
	if children {
		for _, child := range node.Items {
			child.Print(children, w)
		}
	}
}

func (node *Todoitem) Printer(w io.Writer, printer func(node *Todoitem, w io.Writer)) {
	printer(node, w)

	for _, child := range node.Items {
		child.Printer(w, printer)
	}
}

func (node *Todoitem) SetOffset(ofs int) {
	node.offset = ofs
}

func (node *Todoitem) Offset() int {
	return node.offset
}

func getParent(node *Todoitem, file *Todofile) *Todoitem {
	if node == nil {
		return nil
	}

	return node.parent
}

func parentAdd(parent, node *Todoitem, file *Todofile) {
	if parent == nil {
		file.Items = append(file.Items, node)
	} else {
		parent.Add(node)
	}
}

func parseNode(prev *Todoitem, parent *Todoitem, line string, file *Todofile) (*Todoitem, *Todoitem, error) {
	idt, node, err := parseLine(line)
	if err != nil {
		return parent, nil, err
	}

	if prev == nil {
		parentAdd(nil, &node, file)
		return parent, &node, nil
	}

	if idt > prev.Ident {
		prev.Add(&node)
		parent = prev
	} else if idt < prev.Ident {
		parent = getParent(parent, file)
		parentAdd(parent, &node, file)
	} else {
		parentAdd(parent, &node, file)
	}

	return parent, &node, nil
}

var (
	box    = `- ❍ ❑ ■ ⬜ □ ☐ ▪ ▫ – — ≡ → › []`
	done   = `✔ ✓ ☑ + [x] [X] [+]`
	cancel = `✘ x X [-]`
)

const Space = " "

func enc(s string) string {
	var (
		r  = []rune(s)
		rs = make([]rune, 0)
	)

	for _, c := range r {
		switch c {
		case '+', '[', ']':
			rs = append(rs, '\\', c)
		default:
			rs = append(rs, c)
		}
	}

	return string(rs)
}

func buildLineRegexp() *regexp.Regexp {
	ss := strings.Split(enc(box), Space)
	ss = append(ss, `\[\s+\]`)
	dss := strings.Split(enc(done), Space)
	ss = append(ss, dss...)
	css := strings.Split(enc(cancel), Space)
	ss = append(ss, css...)
	s := strings.Join(ss, "|")
	s = "^(" + s + ")\\s+(.*?)$"

	return regexp.MustCompile(s)
}

func buildTitleRegexp() *regexp.Regexp {
	ss := strings.Split(enc(box), Space)
	dss := strings.Split(enc(done), Space)
	ss = append(ss, dss...)
	css := strings.Split(enc(cancel), Space)
	ss = append(ss, css...)
	s := strings.Join(ss, "")
	s = "^[^" + s + "].*?:$"

	return regexp.MustCompile(s)
}

var (
	reLine  = buildLineRegexp()
	reTitle = buildTitleRegexp()
)

func tokenStatus(token string) ItemStatus {
	ss := strings.Split(box, " ")
	ss = append(ss, `[ ]`)
	for _, s := range ss {
		if s == token {
			return StPending
		}
	}
	ss = strings.Split(done, " ")
	for _, s := range ss {
		if s == token {
			return StDone
		}
	}

	ss = strings.Split(cancel, " ")
	for _, s := range ss {
		if s == token {
			return StCancel
		}
	}

	return StPending
}

func ident(line string, width int) (int, string) {
	var (
		r = []rune(line)
		j int
	)

	for i, c := range r {
		switch c {
		case ' ':
			j++
		case '\t':
			j += width
		default:
			return j, string(r[i:])
		}
	}

	return j, ""
}

func parseLine(line string) (int, Todoitem, error) {
	var (
		todo          Todoitem
		idt, pureline = ident(line, 4)
	)

	s := reLine.FindStringSubmatch(pureline)
	if len(s) > 0 {
		todo.Type = ItItem
		todo.Token = s[1]
		todo.Text = s[2]
		todo.Status = tokenStatus(s[1])
		goto Exit
	}

	s = reTitle.FindStringSubmatch(pureline)
	if len(s) > 0 {
		todo.Type = ItTitle
		todo.Status = StUnknown
		todo.Text = s[0]
		goto Exit
	}

	todo.Text = pureline

Exit:
	todo.Ident = idt
	todo.Items = make([]*Todoitem, 0)
	parseTag(todo.Text, &todo)
	parseFormatText(todo.Text, &todo)

	return idt, todo, nil
}

var (
	reTag    = regexp.MustCompile(`@[\w\p{L}\d]+(\([\w\d\s:-]+\)|\S)`)
	reFmtTxt = regexp.MustCompile("\\*.*?\\*|\\`.*?\\`|~.*?~|_.*?_")
)

func tagType(tag string) TagType {
	if tag[0] == '@' {
		tag = tag[1:]
	} else {
		return TagNormal
	}

	if strings.HasPrefix(tag, "done") {
		return TagDone
	} else if strings.HasPrefix(tag, "started") {
		return TagStarted
	} else if strings.HasPrefix(tag, "est") {
		return TagEst
	} else if strings.HasPrefix(tag, "lasted") {
		return TagLasted
	} else if strings.HasPrefix(tag, "critical") {
		return TagCritical
	} else if strings.HasPrefix(tag, "high") {
		return TagHigh
	} else if strings.HasPrefix(tag, "low") {
		return TagLow
	} else if strings.HasPrefix(tag, "today") {
		return TagToday
	}

	return TagNormal
}

func parseTag(text string, node *Todoitem) []Tag {
	var tags = make([]Tag, 0)
	idxs := reTag.FindAllStringIndex(text, -1)
	if len(idxs) == 0 {
		return nil
	}

	for _, idx := range idxs {
		i0, i1 := idx[0], idx[1]
		tag := Tag{
			Start: i0,
			Stop:  i1,
			Type:  tagType(text[i0:i1]),
			Text:  text[i0:i1],
		}
		tags = append(tags, tag)
	}

	node.Tags = append(node.Tags, tags...)
	return tags
}

func formatTag(text string) TagType {
	switch text[0] {
	case '*':
		return TagBold
	case '_':
		return TagItalic
	case '~':
		return TagDeleted
	case '`':
		return TagCode
	}
	return TagUnknown
}
func parseFormatText(text string, node *Todoitem) []Tag {
	var tags = make([]Tag, 0)
	idxs := reFmtTxt.FindAllStringIndex(text, -1)
	if len(idxs) == 0 {
		return nil
	}

	for _, idx := range idxs {
		i0, i1 := idx[0], idx[1]
		tag := Tag{
			Start: i0,
			Stop:  i1,
			Type:  formatTag(text[i0:i1]),
			Text:  text[i0:i1],
		}
		tags = append(tags, tag)
	}

	node.Tags = append(node.Tags, tags...)

	return tags
}
