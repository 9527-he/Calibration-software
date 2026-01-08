/*
@Author: zzh
@Date: 2025-09-19 14:00:47
@LastEditors: zzh
@LastEditTime: 2025-12-08
@Description: 导航栏窗口 - 校准工具简化版
*/
package window

import (
	"fmt" // 添加此行
	"main/config"
	"main/devices/pdu"
	"main/devices/power_source"
	"math"
	"strings"
	"time"

	"main/gui/customWidget"
	"main/gui/table"
	"main/types"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type Navigate string

func (n Navigate) String() string { return string(n) }

var (
	navigateTree          *widget.Tree
	standardSourceRunning          = false
	NavCalibration        Navigate = "calibration"
)

func MakeNavigate(setTutorial func(tutorial Tutorial), parent fyne.Window) fyne.CanvasObject {

	navigateTree = &widget.Tree{
		ChildUIDs: func(uid string) []string {
			if uid != "" {
				return []string{}
			}
			return []string{NavCalibration.String()}
		},

		IsBranch: func(uid string) bool {
			return uid == ""
		},

		CreateNode: func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("校准工具")
		},

		UpdateNode: func(uid string, branch bool, obj fyne.CanvasObject) {
			tutorial, ok := Tutorials[Navigate(uid)]
			if !ok {
				fyne.LogError("Missing tutorial panel: "+uid, nil)
				return
			}
			obj.(*widget.Label).SetText(tutorial.Title)
		},

		OnSelected: func(uid string) {
			if tutorial, ok := Tutorials[Navigate(uid)]; ok {
				setTutorial(tutorial)
			}
		},
	}
	SelectNavigate(NavCalibration)
	// 创建标准源控制按钮（先创建按钮，再设置回调以支持互相引用）
	btnStart := widget.NewButton("开启标准源", nil)
	btnStop := widget.NewButton("关闭标准源", nil)
	set_single := widget.NewButton("设置标准源", nil)
	set_correction := widget.NewButton("校准", nil)
	Test_oracle1 := widget.NewButton("测试1", nil)
	Test_oracle2 := widget.NewButton("测试2", nil)
	reset_button := widget.NewButton("底数清零", nil)

	reset_button.OnTapped = func() {
		go func() {
			var win fyne.Window
			if a := fyne.CurrentApp(); a != nil {
				ws := a.Driver().AllWindows()
				if len(ws) > 0 {
					win = ws[0]
				}
			}

			if Info.PDUCOM == "" || Info.PDUCOM == "无" {
				SetNoteText("错误: PDU串口未选择")
				customWidget.NoBlockShowDialog("错误", "PDU串口未选择", "no.png", win)
				return
			}

			openedHere := false
			if err := pdu.Open(Info.PDUCOM); err != nil {
				if !strings.Contains(strings.ToLower(err.Error()), "device already open") {
					errMsg := fmt.Sprintf("打开PDU失败: %v", err)
					SetNoteText(errMsg)
					customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
					return
				}
			} else {
				openedHere = true
			}

			SetNoteText("发送清零命令到PDU...")
			resp, err := pdu.SendCommandByName("clear_energy")
			if err != nil {
				errMsg := fmt.Sprintf("发送清零命令失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("清零", errMsg, "no.png", win)
				if openedHere {
					pdu.Close()
				}
				return
			}

			if len(resp) == 0 {
				SetNoteText("清零: 无响应")
				customWidget.NoBlockShowDialog("清零", "清零: 无响应", "no.png", win)
				if openedHere {
					pdu.Close()
				}
				return
			}

			// 简单异或校验（若响应含校验尾字节）
			if len(resp) >= 2 {
				chk := byte(0)
				for i := 0; i < len(resp)-1; i++ {
					chk ^= resp[i]
				}
				if chk != resp[len(resp)-1] {
					SetNoteText("清零响应校验失败")
					customWidget.NoBlockShowDialog("清零", "清零响应校验失败", "no.png", win)
					if openedHere {
						pdu.Close()
					}
					return
				}
			}

			SetNoteText("PDU清零命令已发送并收到有效应答")
			customWidget.NoBlockShowDialog("清零", "清零成功", "yes.png", win)

			if openedHere {
				pdu.Close()
			}
		}()
	}

	Test_oracle2.OnTapped = func() {
		go func() {
			// 执行独立的 3A 测试（不包含校准步骤）
			var win fyne.Window
			if a := fyne.CurrentApp(); a != nil {
				ws := a.Driver().AllWindows()
				if len(ws) > 0 {
					win = ws[0]
				}
			}
			if Info.SourceCOM == "" || Info.SourceCOM == "无" {
				SetNoteText("错误: 标准电源串口未选择")
				customWidget.NoBlockShowDialog("错误", "标准电源串口未选择", "no.png", win)
				return
			}
			if Info.PDUCOM == "" || Info.PDUCOM == "无" {
				SetNoteText("错误: PDU串口未选择")
				customWidget.NoBlockShowDialog("错误", "PDU串口未选择", "no.png", win)
				return
			}

			// 打开设备
			if err := power_source.Open(Info.SourceCOM); err != nil {
				errMsg := fmt.Sprintf("打开标准电源失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
				return
			}
			defer power_source.Close()

			if err := pdu.Open(Info.PDUCOM); err != nil {
				errMsg := fmt.Sprintf("打开PDU失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
				return
			}
			defer pdu.Close()

			SetNoteText("设置标准电源为 200V 3A 并启动输出...")
			src, err := power_source.ConfigureAndVerify("single_200_3")
			if err != nil {
				errMsg := fmt.Sprintf("设置标准电源失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
				return
			}
			_, err = power_source.ConfigureAndVerify("start_L1")
			if err != nil {
				errMsg := fmt.Sprintf("启动标准电源失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
				return
			}
			time.Sleep(3 * time.Second)

			pduData, err := pdu.ReadData()
			if err != nil {
				errMsg := fmt.Sprintf("读取PDU失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
				return
			}

			// 构造数据并判断
			t := types.CalibrationData{TestCurrent: 3.0}
			t.SourceVoltage = src.Voltage
			t.SourceCurrent = src.Current
			t.SourcePower = src.Power
			t.SourcePF = src.PF
			t.PDUVoltage = pduData.Voltage
			t.PDUCurrent = pduData.Current
			t.PDUPower = pduData.Power
			t.PDUPF = pduData.PF

			t.VoltageDiff = math.Abs(t.PDUVoltage - t.SourceVoltage)
			t.CurrentDiff = math.Abs(t.PDUCurrent - t.SourceCurrent)
			t.PowerDiff = math.Abs(t.PDUPower - t.SourcePower)
			t.PFDiff = math.Abs(t.PDUPF - t.SourcePF)

			pass := t.VoltageDiff <= config.Thresholds.VMaxDiff &&
				t.CurrentDiff <= config.Thresholds.IMaxDiff &&
				t.PowerDiff <= config.Thresholds.PMaxDiff &&
				t.PFDiff <= config.Thresholds.PfMaxDiff

			// 更新表格并弹窗
			table.UpdateCalibrationTable(append([]types.CalibrationData{}, t))
			if pass {
				customWidget.NoBlockShowDialog("测试结果", "3A测试: PASS", "yes.png", win)
			} else {
				customWidget.NoBlockShowDialog("测试结果", "3A测试: NG", "no.png", win)
			}

		}()
	}

	Test_oracle1.OnTapped = func() {
		go func() {
			// 执行独立的 6A 测试（不包含校准步骤）
			var win fyne.Window
			if a := fyne.CurrentApp(); a != nil {
				ws := a.Driver().AllWindows()
				if len(ws) > 0 {
					win = ws[0]
				}
			}
			if Info.SourceCOM == "" || Info.SourceCOM == "无" {
				SetNoteText("错误: 标准电源串口未选择")
				customWidget.NoBlockShowDialog("错误", "标准电源串口未选择", "no.png", win)
				return
			}
			if Info.PDUCOM == "" || Info.PDUCOM == "无" {
				SetNoteText("错误: PDU串口未选择")
				customWidget.NoBlockShowDialog("错误", "PDU串口未选择", "no.png", win)
				return
			}

			if err := power_source.Open(Info.SourceCOM); err != nil {
				errMsg := fmt.Sprintf("打开标准电源失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
				return
			}
			defer power_source.Close()

			if err := pdu.Open(Info.PDUCOM); err != nil {
				errMsg := fmt.Sprintf("打开PDU失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
				return
			}
			defer pdu.Close()

			SetNoteText("设置标准电源为 220V 6A 并启动输出...")
			src, err := power_source.ConfigureAndVerify("single_220_6")
			if err != nil {
				errMsg := fmt.Sprintf("设置标准电源失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
				return
			}
			_, err = power_source.ConfigureAndVerify("start_L1")
			if err != nil {
				errMsg := fmt.Sprintf("启动标准电源失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
				return
			}
			time.Sleep(3 * time.Second)

			pduData, err := pdu.ReadData()
			if err != nil {
				errMsg := fmt.Sprintf("读取PDU失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
				return
			}

			t := types.CalibrationData{TestCurrent: 6.0}
			t.SourceVoltage = src.Voltage
			t.SourceCurrent = src.Current
			t.SourcePower = src.Power
			t.SourcePF = src.PF
			t.PDUVoltage = pduData.Voltage
			t.PDUCurrent = pduData.Current
			t.PDUPower = pduData.Power
			t.PDUPF = pduData.PF

			t.VoltageDiff = math.Abs(t.PDUVoltage - t.SourceVoltage)
			t.CurrentDiff = math.Abs(t.PDUCurrent - t.SourceCurrent)
			t.PowerDiff = math.Abs(t.PDUPower - t.SourcePower)
			t.PFDiff = math.Abs(t.PDUPF - t.SourcePF)

			pass := t.VoltageDiff <= config.Thresholds.VMaxDiff &&
				t.CurrentDiff <= config.Thresholds.IMaxDiff &&
				t.PowerDiff <= config.Thresholds.PMaxDiff &&
				t.PFDiff <= config.Thresholds.PfMaxDiff

			table.UpdateCalibrationTable(append([]types.CalibrationData{}, t))
			if pass {
				customWidget.NoBlockShowDialog("测试结果", "6A测试: PASS", "yes.png", win)
			} else {
				customWidget.NoBlockShowDialog("测试结果", "6A测试: NG", "no.png", win)
			}

		}()
	}

	set_correction.OnTapped = func() {
		go func() {
			// 发送 PDU 校准命令（使用 configs/serial.yaml 中的 "calibrate" 字节序列）
			var win fyne.Window
			if a := fyne.CurrentApp(); a != nil {
				ws := a.Driver().AllWindows()
				if len(ws) > 0 {
					win = ws[0]
				}
			}
			if Info.PDUCOM == "" || Info.PDUCOM == "无" {
				SetNoteText("错误: PDU串口未选择")
				customWidget.NoBlockShowDialog("错误", "PDU串口未选择", "no.png", win)
				return
			}

			if err := pdu.Open(Info.PDUCOM); err != nil {
				errMsg := fmt.Sprintf("打开PDU失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
				return
			}
			defer pdu.Close()

			SetNoteText("发送校准命令...")
			resp, err := pdu.SendCommandByName("calibrate")
			if err != nil {
				errMsg := fmt.Sprintf("发送校准命令失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("校准", errMsg, "no.png", win)
				return
			}
			if len(resp) == 0 {
				SetNoteText("校准: 无响应")
				customWidget.NoBlockShowDialog("校准", "校准: 无响应", "no.png", win)
				return
			}

			// 简单异或校验检查（若有尾校验字节）
			if len(resp) >= 2 {
				chk := byte(0)
				for i := 0; i < len(resp)-1; i++ {
					chk ^= resp[i]
				}
				if chk != resp[len(resp)-1] {
					SetNoteText("校准响应校验失败")
					customWidget.NoBlockShowDialog("校准", "校准响应校验失败", "no.png", win)
					return
				}
			}

			SetNoteText("校准命令发送并收到应答")
			customWidget.NoBlockShowDialog("校准", "校准命令发送并收到应答", "yes.png", win)

		}()
	}

	// 点击发送 single_220_6 原始命令并打印回包（对比成功/失败响应包）
	set_single.OnTapped = func() {
		go func() {
			var win fyne.Window
			if a := fyne.CurrentApp(); a != nil {
				ws := a.Driver().AllWindows()
				if len(ws) > 0 {
					win = ws[0]
				}
			}
			if Info.SourceCOM == "" || Info.SourceCOM == "无" {
				SetNoteText("错误: 标准电源串口未选择")
				customWidget.NoBlockShowDialog("错误", "标准电源串口未选择", "no.png", win)
				return
			}

			openedHere := false
			if err := power_source.Open(Info.SourceCOM); err != nil {
				// 如果是已打开则继续，否则报错
				if !strings.Contains(strings.ToLower(err.Error()), "device already open") {
					errMsg := fmt.Sprintf("打开标准电源失败: %v", err)
					SetNoteText(errMsg)
					customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
					return
				}
			} else {
				openedHere = true
			}

			resp, err := power_source.SendCommandByName("single_220_6")
			if err != nil {
				errMsg := fmt.Sprintf("发送命令失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
			} else if len(resp) == 0 {
				// 无响应处理
				errMsg := "发送命令: 无响应"
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
			} else {
				SetNoteText("已发送: 设置电压220V电流6A 命令")
				fmt.Printf("single_220_6 实际响应 bytes: % X\n", resp)
			}

			if openedHere {
				power_source.Close()
			}
		}()
	}

	// 设置按钮回调 - 两个按钮都始终可用（独立按键设计）
	btnStart.OnTapped = func() {
		var win fyne.Window
		if a := fyne.CurrentApp(); a != nil {
			ws := a.Driver().AllWindows()
			if len(ws) > 0 {
				win = ws[0]
			}
		}

		if Info.SourceCOM == "" || Info.SourceCOM == "无" {
			SetNoteText("错误: 请先选择标准电源串口")
			customWidget.NoBlockShowDialog("错误", "标准电源串口未选择", "no.png", win)
			return
		}

		if Info.PDUCOM == "" || Info.PDUCOM == "无" {
			SetNoteText("错误: 请先选择电源模块串口")
			customWidget.NoBlockShowDialog("错误", "电源模块串口未选择", "no.png", win)
			return
		}

		if err := power_source.Open(Info.SourceCOM); err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "device already open") {
				errMsg := fmt.Sprintf("打开标准电源失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
				return
			}
		}

		// 打开电源模块串口并保持打开，供后续读取使用
		if err := pdu.Open(Info.PDUCOM); err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "device already open") {
				errMsg := fmt.Sprintf("打开电源模块失败：%V", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
				return
			}
		}

		standardSourceRunning = true

		go fyne.Do(func() {
			SetNoteText("启动标准电源输出...")
			_, err := power_source.ConfigureAndVerify("start_L1")
			if err != nil {
				fmt.Printf("启动标准源失败: %v\n", err)
				standardSourceRunning = false
				errMsg := fmt.Sprintf("启动标准电源失败: %v", err)
				SetNoteText(errMsg)
				var win fyne.Window
				if a := fyne.CurrentApp(); a != nil {
					ws := a.Driver().AllWindows()
					if len(ws) > 0 {
						win = ws[0]
					}
				}

				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
			} else {

				time.Sleep(3 * time.Second)

				pduData, err := pdu.ReadData()
				if err != nil {
					errMsg := fmt.Sprintf("读取PDU失败: %v", err)
					SetNoteText(errMsg)
					customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
					return
				}

				src, err := power_source.ConfigureAndVerify("single_220_6")
				if err != nil {
					errMsg := fmt.Sprintf("设置标准电源失败: %v", err)
					SetNoteText(errMsg)
					customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
					return
				}

				t := types.CalibrationData{TestCurrent: 6.0}
				t.SourceVoltage = src.Voltage
				t.SourceCurrent = src.Current
				t.SourcePower = src.Power
				t.SourcePF = src.PF

				t.PDUVoltage = pduData.Voltage
				t.PDUCurrent = pduData.Current
				t.PDUPower = pduData.Power
				t.PDUPF = pduData.PF
				table.UpdateCalibrationTable(append([]types.CalibrationData{}, t))
				SetNoteText("标准电源已启动")

				// 启动完毕后,关闭设备连接,不占用串口资源,影响其他按键调用串口
				defer pdu.Close()
				defer power_source.Close()
			}
		})
	}

	btnStop.OnTapped = func() {
		standardSourceRunning = false
		var win fyne.Window
		if a := fyne.CurrentApp(); a != nil {
			ws := a.Driver().AllWindows()
			if len(ws) > 0 {
				win = ws[0]
			}
		}
		if Info.SourceCOM == "" || Info.SourceCOM == "无" {
			SetNoteText("错误: 请先选择标准电源串口")
			customWidget.NoBlockShowDialog("错误", "标准电源串口未选择", "no.png", win)
			return
		}

		// 打开串口（如果还没打开的话）
		if err := power_source.Open(Info.SourceCOM); err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "device already open") {
				errMsg := fmt.Sprintf("打开标准电源失败: %v", err)
				SetNoteText(errMsg)
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
				return
			}
		}

		go fyne.Do(func() {
			SetNoteText("关闭标准电源输出...")
			_, err := power_source.ConfigureAndVerify("three_stop")
			if err != nil {
				fmt.Printf("关闭标准源失败: %v\n", err)
				errMsg := fmt.Sprintf("关闭标准电源失败: %v", err)
				SetNoteText(errMsg)
				var win fyne.Window
				if a := fyne.CurrentApp(); a != nil {
					ws := a.Driver().AllWindows()
					if len(ws) > 0 {
						win = ws[0]
					}
				}
				customWidget.NoBlockShowDialog("错误", errMsg, "no.png", win)
			} else {
				SetNoteText("标准电源已关闭")
				// 清空表格显示
				t := types.CalibrationData{SourceVoltage: 0.0}
				t.PDUVoltage = 0.0
				t.PDUCurrent = 0.0
				t.PDUPower = 0.0
				t.PDUPF = 0.0
				table.UpdateCalibrationTable(append([]types.CalibrationData{}, t))
			}
			// 关闭本地串口连接
			power_source.Close()
		})
	}

	// 底部水平容器：两个按钮并排，居中显示
	bottomContainer := container.NewVBox(
		layout.NewSpacer(),
		set_single,
		btnStart,
		set_correction,
		Test_oracle1,
		Test_oracle2,
		reset_button,
		btnStop,

		layout.NewSpacer(),
	)

	// 将按钮放到最右侧，导航树在中间
	return container.NewBorder(nil, bottomContainer, nil, nil, navigateTree)

}

func SelectNavigate(nav Navigate) {
	fyne.Do(func() {
		navigateTree.Select(string(nav))
	})
}
