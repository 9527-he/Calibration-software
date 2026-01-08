/*
 * @Author: zzh
 * @Date: 2025-09-28 17:07:36
 * @LastEditors: zzh
 * @LastEditTime: 2025-09-30 12:15:25
 * @Description: 固定宽度的 Label
 */
package customWidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type fixedLabel struct {
	widget.Label
	width float32
}

func NewFixedLabel(text string, w float32) *fixedLabel {
	l := &fixedLabel{width: w}
	l.ExtendBaseWidget(l)
	l.SetText(text)
	return l
}

func (l *fixedLabel) MinSize() fyne.Size {
	return fyne.NewSize(l.width, l.Label.MinSize().Height)
}
