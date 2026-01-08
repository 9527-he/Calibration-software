/*
 * @Author: zzh
 * @Date: 2025-09-19 10:43:02
 * @LastEditors: zzh
 * @LastEditTime: 2025-11-19 13:52:37
 * @Description: 整个 Gui 的运行
 */
package gui

import (
	_ "embed"
	"main/gui/table"
	"main/gui/window"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
)

var App fyne.App
var Window fyne.Window

//go:embed icon/hpx.ico
var logoBytes []byte

// Gui Run 必须在主线程执行
func Run(title string, onStarted func()) {
	res := fyne.NewStaticResource("hpx.ico", logoBytes)

	App = app.NewWithID("io.fyne.demo")
	App.SetIcon(res)
	// onStarted 将会在 Gui 跑起来之后被执行
	App.Lifecycle().SetOnStarted(onStarted)

	Window = App.NewWindow(title)
	content := container.NewStack()

	// 函数指针 将会在导航栏被点击时调用
	setTutorial := func(tutorial window.Tutorial) {
		content.Objects = []fyne.CanvasObject{tutorial.View()}
		content.Refresh()
	}
	// 空容器 上面的回调就会往这个容器内填充内容(表格)
	tutorial := container.NewBorder(nil, nil, nil, nil, content)

	table.Make()
	// 创建水平容器 并设置比例 也就是在设置导航栏 表格栏 信息栏 的比例
	split := container.NewHSplit(window.MakeNavigate(setTutorial, Window), tutorial)
	split.Offset = 0.1

	split = container.NewHSplit(split, window.MakeInfoWindow(Window))
	split.Offset = 0.9

	Window.SetContent(split)
	Window.Resize(fyne.NewSize(1000, 600))

	Window.ShowAndRun()
}
