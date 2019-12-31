package printer

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/hysios/todo/parser"
	"github.com/imdario/mergo"
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
)

type Color = color.Attribute

type Printer struct {
	Palette map[ItemType][]Color

	todofile *parser.Todofile
}

var defaultPalette = map[ItemType][]Color{
	ClBase:    []Color{color.FgWhite},
	ClItem:    []Color{color.FgWhite},
	ClTitle:   []Color{color.FgCyan},
	ClText:    []Color{color.Faint},
	ClDone:    []Color{color.FgGreen},
	ClCancel:  []Color{color.FgRed},
	ClTag:     []Color{color.FgYellow},
	ClCustom1: []Color{color.BgHiRed, color.FgBlack},
	ClCustom2: []Color{color.BgHiCyan, color.FgBlack},
	ClCustom3: []Color{color.BgYellow, color.FgBlack},
	ClCustom4: []Color{color.BgMagenta, color.FgBlack},
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
	default:
		return ClTag
	}
}

func shiftTag(tags []parser.Tag) (*parser.Tag, []parser.Tag) {
	if len(tags) == 0 {
		return nil, nil
	}

	tag := &tags[0]
	tags = tags[1:]
	return tag, tags
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

func (print *Printer) colorWithTags(text string, c *color.Color, tags []parser.Tag) string {
	var full string
	if len(tags) == 0 {
		return c.Sprint(text)
	}

	tag, tags := shiftTag(tags)

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
			tag, tags = shiftTag(tags)
		} else if i < tag.Start { // ear tag
			k = tag.Start
			i = k
			full += c.Sprint(text[j:k])
		} else if i > tag.Stop { // after tag
			tag, tags = shiftTag(tags) // pop tag
			j = i
		}
	}

	return full
}

func (print *Printer) Print() {
	var sb strings.Builder

	for _, child := range print.todofile.Items {
		child.Printer(&sb, func(node *parser.Todoitem, w io.Writer) {
			ctxt := print.pickColor(typeClr(node.Type))
			cStat := print.pickColor(statuClr(node.Status))
			if node.Status == parser.StDone || node.Status == parser.StCancel {
				ctxt = cStat
			}

			if node.Type == parser.ItItem {
				mainText := print.colorWithTags(node.Text, ctxt, node.Tags)
				fmt.Fprintf(w, "%s%s %s\n", strings.Repeat(" ", node.Ident), cStat.Sprint(node.Token), mainText)
			} else {
				fmt.Fprintf(w, "%s%s\n", strings.Repeat(" ", node.Ident), ctxt.Sprint(node.Text))
			}
		})
	}

	fmt.Println(sb.String())
}
