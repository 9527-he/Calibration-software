/*
 * @Author: zzh
 * @Date: 2025-09-26 11:30:12
 * @LastEditors: zzh
 * @LastEditTime: 2025-11-19 13:49:23
 * @Description: 将 标签和选择框 组合为一个控件
 */
package customWidget

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type wideSelect struct {
	widget.Select
	minW float32
}

func newWideSelect(minW float32, options []string, callback func(string)) *wideSelect {
	e := &wideSelect{minW: minW}
	e.ExtendBaseWidget(e)
	e.OnChanged = callback
	e.Options = options
	return e
}

func (e *wideSelect) MinSize() fyne.Size {
	return fyne.NewSize(e.minW, e.Select.MinSize().Height)
}

func (e *wideSelect) Disable() {
	e.Select.Disable()
}

func (e *wideSelect) Enable() {
	e.Select.Enable()
}

type SelectGroup struct {
	widget.BaseWidget
	Label  *fixedLabel
	Select *wideSelect
}

func NewSelectGroup(text string, options []string, callback func(string)) *SelectGroup {
	g := &SelectGroup{}
	g.ExtendBaseWidget(g)
	g.Label = NewFixedLabel(text, 70)
	g.Select = newWideSelect(160, options, callback)
	g.Select.PlaceHolder = "请选择"
	return g
}

func (g *SelectGroup) CreateRenderer() fyne.WidgetRenderer {
	gap := canvas.NewRectangle(color.Transparent)
	gap.SetMinSize(fyne.NewSize(10, 0))
	return widget.NewSimpleRenderer(container.NewHBox(g.Label, gap, g.Select))
}

func (e *SelectGroup) Disable() {
	e.Select.Disable()
}

func (e *SelectGroup) Enable() {
	e.Select.Enable()
}
