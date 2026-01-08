/*
 * @Author: zzh
 * @Date: 2025-09-26 11:30:12
 * @LastEditors: zzh
 * @LastEditTime: 2025-11-19 13:49:29
 * @Description: 将 标签和输入框 组合为一个控件 
 */
package customWidget

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type wideEntry struct {
	widget.Entry
	minW float32
}

func newWideEntry(target *string, minW float32) *wideEntry {
	e := &wideEntry{minW: minW}
	e.ExtendBaseWidget(e)
	e.OnChanged = func(s string) {
		*target = s
	}
	return e
}

func (e *wideEntry) MinSize() fyne.Size {
	return fyne.NewSize(e.minW, e.Entry.MinSize().Height)
}

type InputGroup struct {
	widget.BaseWidget
	Label *fixedLabel
	Entry *wideEntry
}

func NewInputGroup(text string, target *string) *InputGroup {
	g := &InputGroup{}
	g.ExtendBaseWidget(g)
	g.Label = NewFixedLabel(text, 70)
	g.Entry = newWideEntry(target, 160)
	return g
}

func (g *InputGroup) CreateRenderer() fyne.WidgetRenderer {
	gap := canvas.NewRectangle(color.Transparent)
	gap.SetMinSize(fyne.NewSize(10, 0))
	return widget.NewSimpleRenderer(container.NewHBox(g.Label, gap, g.Entry))
}

func (e *InputGroup) Disable() {
	e.Entry.Disable()
}

func (e *InputGroup) Enable() {
	e.Entry.Enable()
}
