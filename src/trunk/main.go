/*
 * @Author: zzh
 * @Date: 2025-09-09 16:18:24
 * @LastEditors:
 * @LastEditTime: 2025-12-09 09:25:44
 * @Description: PDU校准工具主程序
 */
package main

import (
	"fmt"
	"main/check"
	"main/config"
	"main/gui"
	"main/gui/customWidget"
	"main/gui/window"
	"main/terminal"
	"strings"
	"time"
)

// From build.bat
var (
	version   string
	buildTime string
	buildDate string
)

func main() {
	gui_title := fmt.Sprint("PDU 校准工具 V", version, " Publish: ", buildDate, " ", buildTime)
	config.Load()
	gui.Run(gui_title, run)
}

func run() {
	startTime := time.Now()
	go func() {
		for {
			window.SetMultiFuncButtonText("开始校准")
			window.WaitClickMultiFuncButton()
			sn, ok := waitInputSNandModel()
			if !ok {
				continue
			}
			fmt.Println("当前设备: " + sn)

			// 设置串口号
			if !setupSerialPorts() {
				continue
			}

			fmt.Println("=== 开始校准流程 ===")

			if err := check.Run(startTime); err != nil {
				fmt.Println("校准失败:", err)
			}

			fmt.Println("=== 校准流程结束 ===")
		}
	}()
}

func setupSerialPorts() bool {
	// 直接使用界面上的下拉选择结果
	if window.Info.SourceCOM == "" || window.Info.SourceCOM == "无" {
		customWidget.ShowDialog("错误", "请选择标准电源串口", "no.png", gui.Window)
		return false
	}
	if window.Info.PDUCOM == "" || window.Info.PDUCOM == "无" {
		customWidget.ShowDialog("错误", "请选择PDU串口", "no.png", gui.Window)
		return false
	}

	fmt.Printf("标准电源串口: %s, PDU串口: %s\n", window.Info.SourceCOM, window.Info.PDUCOM)
	return true
}

func waitInputSNandModel() (sn string, ok bool) {
	sn = customWidget.ShowInputDialog("请输入 SN 码", "", gui.Window)
	sn = strings.ToUpper(sn)
	if nil != terminal.CheckSN(sn) {
		customWidget.ShowDialog("错误", "SN 码格式错误", "no.png", gui.Window)
		return "", false
	}
	window.SnUpdate(sn)
	return sn, true
}
