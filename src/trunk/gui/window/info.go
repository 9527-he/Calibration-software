/*
 * @Author: zzh
 * @Date: 2025-09-23 09:29:23
 * @LastEditors: zzh
 * @LastEditTime: 2025-12-08
 * @Description: 信息窗口 - 校准工具简化版
 */
package window

import (
	"fmt"
	"image/color"
	"main/devices/pdu"
	"main/devices/power_source"
	"main/gui/customWidget"
	"main/util"
	"strings"
	"sync/atomic"

	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"fyne.io/fyne/v2/data/binding"
)

type TestInfo struct {
	SourceCOM string // 标准电源串口
	PDUCOM    string // PDU串口
	Sn        string
}

var (
	Info                  = &TestInfo{}
	nextButtonPress int32 = 0
	sn                    = binding.NewString()

	noteLable       *widget.Label
	nextButton      *widget.Button
	Start_Button    *widget.Button
	systemTimeLabel *widget.Label
)

func SnUpdate(newSn string) {
	sn.Set(newSn)
	Info.Sn = newSn
}

// forcedVariant 用于切换深浅色主题（与原实现保持一致）
type forcedVariant struct {
	fyne.Theme
	variant fyne.ThemeVariant
}

func (f *forcedVariant) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return f.Theme.Color(name, f.variant)
}

func MakeInfoWindow(parent fyne.Window) fyne.CanvasObject {
	noteLable = widget.NewLabel("...")
	systemTimeLabel = widget.NewLabel("")

	// 启动一个 goroutine 每秒更新系统时间显示（UI 更新通过 fyne.Do 在主线程安全执行）
	go func() {
		for {
			now := time.Now().Format("2006/01/02 15:04:05")
			fyne.Do(func() {
				if systemTimeLabel != nil {
					systemTimeLabel.SetText(now)
				}
			})
			time.Sleep(time.Second)
		}
	}()

	// 下拉选择：标准电源串口
	sourceSelect := customWidget.NewSelectGroup("标准源串口:\r", []string{"无"}, func(s string) {
		Info.SourceCOM = s
	})
	// 下拉选择：PDU串口
	pduSelect := customWidget.NewSelectGroup("PDU串口: ", []string{"无"}, func(s string) {
		Info.PDUCOM = s
	})

	// 异步刷新本地串口列表
	go func() {
		for {
			comList, err := util.ScanLocalPorts()
			if err == nil && len(comList) != 0 {
				fyne.Do(func() {
					sourceSelect.Select.SetOptions(comList)
					pduSelect.Select.SetOptions(comList)
					// 默认值（若配置文件给了默认串口）
					if Info.SourceCOM != "" {
						sourceSelect.Select.SetSelected(Info.SourceCOM)
					}
					if Info.PDUCOM != "" {
						pduSelect.Select.SetSelected(Info.PDUCOM)
					}
				})
			} else {
				fyne.Do(func() {
					sourceSelect.Select.SetOptions([]string{"无"})
					pduSelect.Select.SetOptions([]string{"无"})
				})
			}
			// 每秒刷新一次
			time.Sleep(1 * time.Second)
		}
	}()

	nextButton = widget.NewButton("开始校准", func() {
		atomic.StoreInt32(&nextButtonPress, 1)
	})

	Start_Button = widget.NewButton("检查串口", func() {
		// 异步执行串口检查，避免阻塞 UI
		go func() {
			SetStartButtonText("检查中...")
			defer SetStartButtonText("检查串口")

			// 检查标准电源串口选择
			if Info.SourceCOM == "" || Info.SourceCOM == "无" {
				SetNoteText("错误: 标准电源串口未选择")
				return
			}
			// 检查PDU串口选择
			if Info.PDUCOM == "" || Info.PDUCOM == "无" {
				SetNoteText("错误: PDU串口未选择")
				return
			}

			// 收集每个设备的检查结果，避免互相覆盖显示
			msgs := make([]string, 0, 4)

			// 尝试打开标准电源串口
			srcErr := power_source.Open(Info.SourceCOM)
			if srcErr != nil {
				msg := parseSerialOpenError(srcErr)
				msgs = append(msgs, fmt.Sprintf("标准电源(%s): %s", Info.SourceCOM, msg))
			} else {
				power_source.Close()
				msgs = append(msgs, fmt.Sprintf("标准电源(%s): 连接成功", Info.SourceCOM))
			}

			// 尝试打开PDU串口
			pduErr := pdu.Open(Info.PDUCOM)
			if pduErr != nil {
				msg := parseSerialOpenError(pduErr)
				msgs = append(msgs, fmt.Sprintf("PDU(%s): %s", Info.PDUCOM, msg))
			} else {
				pdu.Close()
				msgs = append(msgs, fmt.Sprintf("PDU(%s): 连接成功", Info.PDUCOM))
			}

			// 如果两者都能打开，则加入总体成功提示
			if srcErr == nil && pduErr == nil {
				msgs = append(msgs, fmt.Sprintf("串口检查: 标准电源(%s) \n PDU(%s) 连接正常", Info.SourceCOM, Info.PDUCOM))
			}

			// 最终一次性设置提示，避免覆盖
			SetNoteText(strings.Join(msgs, "\n"))
		}()
	})

	// 主题切换按钮（放到信息面板底部右侧）
	darkBtn := widget.NewButton("暗色", func() {
		fyne.CurrentApp().Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantDark})
	})
	lightBtn := widget.NewButton("亮白", func() {
		fyne.CurrentApp().Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantLight})
	})
	themeButtons := container.NewHBox(layout.NewSpacer(), container.NewGridWithColumns(2, darkBtn, lightBtn))

	// 底部：居中显示系统时间，上方放占位等，底部右侧放主题按钮
	bottomTime := container.NewCenter(systemTimeLabel)

	return container.NewVBox(
		widget.NewLabel("PDU校准工具"),
		widget.NewSeparator(),
		sourceSelect,
		pduSelect,
		nextButton,
		Start_Button,
		layout.NewSpacer(),
		noteLable,
		themeButtons,
		bottomTime,
	)
}

func WaitClickMultiFuncButton() {
	atomic.StoreInt32(&nextButtonPress, 0)
	for {
		if atomic.LoadInt32(&nextButtonPress) == 1 {
			return
		}
	}
}

func SetNoteText(text string) {
	fyne.Do(func() {
		noteLable.SetText(text)
	})
}

func SetMultiFuncButtonText(text string) {
	fyne.Do(func() {
		nextButton.SetText(text)
	})
}

func SetStartButtonText(text string) {
	fyne.Do(func() {
		Start_Button.SetText(text)
	})
}

// parseSerialOpenError 将串口打开错误翻译为友好提示
func parseSerialOpenError(err error) string {
	if err == nil {
		return "连接成功\n"
	}
	e := strings.ToLower(err.Error())
	if strings.Contains(e, "device already open") {
		return "已连接\n"
	}
	if strings.Contains(e, "access is denied") || strings.Contains(e, "permission denied") || strings.Contains(e, "access") {
		return "串口被占用:\n" + err.Error()

	}
	return "连接失败:\n " + err.Error()
}

// 更新校准表格数据 (由table包实现)
func UpdateCalibrationTable(data interface{}) {
	// 由 table.UpdateCalibrationTable 处理
	// 在 main/gui/gui.go 中调用 table.UpdateCalibrationTable
}
