/*
 * @Author: zzh
 * @Date: 2025-09-27 08:50:53
 * @LastEditors: zzh
 * @LastEditTime: 2025-12-08
 * @Description: 各种自定义的对话框 - 校准工具简化版
 */
package customWidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// 阻塞式 请勿在 fyne 线程中调用
// 附带两个按钮的对话框 返回值为点击的按钮所对应的文本
func ShowButtonDialog(title, note, opt1, opt2, imgName string, win fyne.Window) string {
	resultChan := make(chan string, 1)
	fyne.Do(func() {
		image := canvas.NewImageFromFile("gui/icon/" + imgName)
		image.FillMode = canvas.ImageFillOriginal

		label := widget.NewLabel(note)
		label.Alignment = fyne.TextAlignCenter

		var dlg dialog.Dialog

		btn1 := widget.NewButton("　　"+opt1+"　　", func() {
			dlg.Hide()
			resultChan <- opt1
		})
		btn2 := widget.NewButton("　　"+opt2+"　　", func() {
			dlg.Hide()
			resultChan <- opt2
		})

		content := container.NewVBox(
			image,
			label,
			container.NewCenter(container.NewHBox(btn1, btn2)),
		)
		dlg = dialog.NewCustomWithoutButtons(title, content, win)
		dlg.Resize(fyne.NewSize(400, 300))
		dlg.Show()
	})

	return <-resultChan
}

// 阻塞式 请勿在 fyne 线程中调用
// 附带文本输入的对话框 返回值为输入的文本
func ShowInputDialog(title, note string, win fyne.Window) string {
	resultChan := make(chan string, 1)
	fyne.Do(func() {
		entry := widget.NewEntry()

		noteLabel := widget.NewLabel(note)
		noteLabel.Alignment = fyne.TextAlignCenter

		var dlg dialog.Dialog
		button := widget.NewButton("　　确认　　", func() {
			if entry.Text != "" {
				dlg.Hide()
				resultChan <- entry.Text
			}
		})

		content := container.NewVBox(
			entry,
			noteLabel,
			button,
		)

		dlg = dialog.NewCustomWithoutButtons(title, content, win)
		dlg.Resize(fyne.NewSize(400, 150))
		dlg.Show()
	})

	return <-resultChan
}

// 阻塞式 请勿在 fyne 线程中调用
// 普通对话框 无返回值 附带的按钮仅用于关闭本对话框
func ShowDialog(title, message, imgName string, win fyne.Window) {
	done := make(chan struct{})
	fyne.Do(func() {
		var contentItems []fyne.CanvasObject

		if imgName != "" {
			image := canvas.NewImageFromFile("gui/icon/" + imgName)
			image.FillMode = canvas.ImageFillOriginal
			contentItems = append(contentItems, image)
		}

		msgLabel := widget.NewLabel(message)
		msgLabel.Alignment = fyne.TextAlignCenter
		contentItems = append(contentItems, msgLabel)

		var dlg dialog.Dialog

		confirmBtn := widget.NewButton("　　确认　　", func() {
			dlg.Hide()
			close(done)
		})
		contentItems = append(contentItems, container.NewCenter(confirmBtn))

		content := container.NewVBox(contentItems...)
		dlg = dialog.NewCustomWithoutButtons(title, content, win)
		if imgName != "" {
			dlg.Resize(fyne.NewSize(400, 250))
		} else {
			dlg.Resize(fyne.NewSize(400, 150))
		}
		dlg.Show()
	})

	<-done
}

// 非阻塞式
// 普通对话框 无返回值 附带的按钮仅用于关闭本对话框
func NoBlockShowDialog(title, message, imgName string, win fyne.Window) {
	fyne.Do(func() {
		var contentItems []fyne.CanvasObject

		if imgName != "" {
			image := canvas.NewImageFromFile("gui/icon/" + imgName)
			image.FillMode = canvas.ImageFillOriginal
			contentItems = append(contentItems, image)
		}

		msgLabel := widget.NewLabel(message)
		msgLabel.Alignment = fyne.TextAlignCenter
		contentItems = append(contentItems, msgLabel)

		var dlg dialog.Dialog
		button := widget.NewButton("　　确认　　", func() {
			dlg.Hide()
		})
		contentItems = append(contentItems, container.NewCenter(button))

		content := container.NewVBox(contentItems...)

		dlg = dialog.NewCustomWithoutButtons(title, content, win)
		if imgName != "" {
			dlg.Resize(fyne.NewSize(400, 250))
		} else {
			dlg.Resize(fyne.NewSize(400, 150))
		}
		dlg.Show()
	})
}
