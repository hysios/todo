package printer

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/hysios/todo/parser"
	"github.com/imdario/mergo"
	"github.com/jinzhu/copier"
)

type Formater interface {
	Print(children bool, w io.Writer)
	Children() []*parser.Todoitem
}

type ItemType int

const (
	ClBase ItemType = iota
	ClItem
	ClTitle
	ClText
	ClDone
	ClCancel
	ClTag
	ClTime
	ClUser
	ClHighlight
	ClCustom1
	ClCustom2
	ClCustom3
	ClCustom4

	ClBold
	ClItalic
	ClDeleted
)

type Color = color.Attribute

type PrinterFunc func(node *parser.Todoitem, w io.Writer)

type Printer struct {
	Palette map[ItemType][]Color

	todofile *parser.Todofile
	pipes    []PrinterFunc
}

var defaultPalette = map[ItemType][]Color{
	ClBase:      []Color{color.FgWhite},
	ClItem:      []Color{color.FgWhite},
	ClTitle:     []Color{color.FgCyan},
	ClText:      []Color{color.Faint},
	ClDone:      []Color{color.FgGreen},
	ClCancel:    []Color{color.FgRed},
	ClTag:       []Color{color.FgYellow},
	ClCustom1:   []Color{color.BgHiRed, color.FgBlack},
	ClCustom2:   []Color{color.BgHiCyan, color.FgBlack},
	ClCustom3:   []Color{color.BgYellow, color.FgBlack},
	ClCustom4:   []Color{color.BgMagenta, color.FgBlack},
	ClBold:      []Color{color.Bold},
	ClItalic:    []Color{color.Italic},
	ClDeleted:   []Color{color.CrossedOut},
	ClHighlight: []Color{color.FgHiYellow},
}

func New(todofile *parser.Todofile) *Printer {
	var p = &Printer{
		todofile: todofile,
		Palette:  make(map[ItemType][]Color),
	}

	mergo.Merge(&p.Palette, defaultPalette)

	return p
}

func typeClr(typ parser.ItemType) ItemType {
	switch typ {
	case parser.ItItem:
		return ClItem
	case parser.ItTitle:
		return ClTitle
	case parser.ItText:
		return ClText
	}

	return ClText
}

func statuClr(status parser.ItemStatus) ItemType {
	switch status {
	case parser.StDone:
		return ClDone
	case parser.StCancel:
		return ClCancel
	default:
		return ClBase
	}
}

func tagClr(tagTyp parser.TagType) ItemType {
	switch tagTyp {
	case parser.TagNormal:
		return ClTag
	case parser.TagCritical:
		return ClCustom1
	case parser.TagHigh:
		return ClCustom2
	case parser.TagLow:
		return ClCustom3
	case parser.TagToday:
		return ClCustom4
	case parser.TagBold:
		return ClBold
	case parser.TagItalic:
		return ClItalic
	case parser.TagDeleted:
		return ClDeleted
	case parser.TagCode:
		return ClHighlight
	default:
		return ClTag
	}
}

func shiftTag(tags []parser.Tag, _opt *colorTagOption) (*parser.Tag, []parser.Tag) {
	if len(tags) == 0 {
		return nil, nil
	}

	var opt colorTagOption
	if _opt != nil {
		opt = *_opt
	}

	tag := tags[0]
	tags = tags[1:]
	tag.Start += opt.Offset
	tag.Stop += opt.Offset
	return &tag, tags
}

func (print *Printer) defaultColor() *color.Color {
	return color.New(defaultPalette[ClBase][0])
}

func (print *Printer) pickColor(typ ItemType) *color.Color {
	var c *color.Color
	if colors, ok := print.Palette[typ]; ok {
		for _, clr := range colors {
			if c == nil {
				c = color.New(clr)
			} else {
				c.Add(clr)
			}
		}
	} else {
		return print.defaultColor()
	}

	return c
}

type colorTagOption struct {
	Offset int
}

func (print *Printer) colorWithTags(text string, c *color.Color, tags []parser.Tag, _opt *colorTagOption) string {
	var (
		full string
		opt  colorTagOption
	)
	if len(tags) == 0 {
		return c.Sprint(text)
	}

	if _opt != nil {
		opt = *_opt
	}

	tag, tags := shiftTag(tags, &opt)

	j, k := 0, 0
	for i := 0; i < len(text); i++ {
		if tag == nil { // not tag
			k = len(text)
			full += c.Sprint(text[j:k])
			return full
		}

		if i >= tag.Start && i <= tag.Stop { // in tag
			j = tag.Start
			k = tag.Stop
			i = tag.Stop - 1
			j = k

			tc := print.pickColor(tagClr(tag.Type))
			full += tc.Sprint(tag.Text)
			tag, tags = shiftTag(tags, &opt)
		} else if i < tag.Start { // ear tag
			k = tag.Start
			i = k
			full += c.Sprint(text[j:k])
		} else if i > tag.Stop { // after tag
			tag, tags = shiftTag(tags, &opt) // pop tag
			j = i
		}
	}

	return full
}

func (print *Printer) Print() {
	var sb strings.Builder

	for _, child := range print.todofile.Items {
		print.printNodePipes(child, &sb)
	}

	fmt.Println(sb.String())
}

func (print *Printer) WriteTo(w io.Writer) {
	for _, child := range print.todofile.Items {
		print.printNodePipesWithoutColor(child, w)
	}

}

func (print *Printer) print(node *parser.Todoitem, w io.Writer) {
	if node.Type == parser.ItItem {
		fmt.Fprintf(w, "%s%s %s\n", strings.Repeat(" ", node.Ident), node.Token, node.Text)
	} else {
		fmt.Fprintf(w, "%s%s\n", strings.Repeat(" ", node.Ident), node.Text)
	}
}

func (print *Printer) printColour(node *parser.Todoitem, w io.Writer) {
	ctxt := print.pickColor(typeClr(node.Type))
	cStat := print.pickColor(statuClr(node.Status))
	if node.Status == parser.StDone || node.Status == parser.StCancel {
		ctxt = cStat
	}

	if node.Type == parser.ItItem {
		mainText := print.colorWithTags(node.Text, ctxt, node.Tags, &colorTagOption{
			Offset: node.Offset(),
		})
		fmt.Fprintf(w, "%s%s %s\n", strings.Repeat(" ", node.Ident), cStat.Sprint(node.Token), mainText)
	} else {
		fmt.Fprintf(w, "%s%s\n", strings.Repeat(" ", node.Ident), ctxt.Sprint(node.Text))
	}
}

func (print *Printer) printNodePipes(node *parser.Todoitem, w io.Writer) {

	node.Printer(w, func(node *parser.Todoitem, w io.Writer) {
		var nnode parser.Todoitem
		copier.Copy(&nnode, node)

		for _, pipe := range print.pipes {
			pipe(&nnode, w)
		}

		print.printColour(&nnode, w)
	})
}

func (print *Printer) printNodePipesWithoutColor(node *parser.Todoitem, w io.Writer) {

	node.Printer(w, func(node *parser.Todoitem, w io.Writer) {
		var nnode parser.Todoitem
		copier.Copy(&nnode, node)

		for _, pipe := range print.pipes {
			pipe(&nnode, w)
		}

		print.print(&nnode, w)
	})
}

func (print *Printer) AddPipe(pipefunc PrinterFunc) {
	if print.pipes == nil {
		print.pipes = make([]PrinterFunc, 0)
	}

	print.pipes = append(print.pipes, pipefunc)
}
