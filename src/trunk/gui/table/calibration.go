/*
 * @Author: zzh
 * @Date: 2025-12-08
 * @Description: 校准工具表格 - 标准源 L1相 | 电源模块 L1相 各含项名与数据列
 */
package table

import (
	"fmt"
	"main/devices/pdu"
	"main/devices/power_source"
	"main/types"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var calibrationTable *widget.Table
var lastCalibrationData []types.CalibrationData
var (
	liveTicker      *time.Ticker
	liveStopChan    chan struct{}
	liveRefreshTick = time.Second
)

func Make() {
	calibrationTable = widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {},
	)
}

func GetCalibrationTable() fyne.CanvasObject {
	if calibrationTable == nil {
		Make()
	}

	// 电源模块完整的 7 项（用于确定行数）
	powerModuleItems := []string{
		"电压(V)",
		"电流 (A)",
		"功率因数(P)",
		"有功功率(W)",
		"无功功率(Var)",
		"有功电能(Wh)",
		"无功电能(VARh)",
	}

	// 标准源的项（只显示基本4项）
	standardItems := []string{
		"电压(V)",
		"电流 (A)",
		"功率因数(P)",
		"有功功率(W)",
	}

	// 行数 = 1（表头） + max(len(standardItems), len(powerModuleItems))
	totalRows := 1 + len(powerModuleItems)
	calibrationTable = widget.NewTable(
		func() (int, int) {
			// 四列：标准源项名 | 标准源L1相数据 | 电源模块项名 | 电源模块L1相数据
			return totalRows, 5
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			label.TextStyle = fyne.TextStyle{} // 清除样式
			// 表头（第一行）
			if id.Row == 0 {
				switch id.Col {
				case 0:
					label.SetText("标准源")
					label.TextStyle.Bold = true
				case 1:
					label.SetText("L1相")
					label.TextStyle.Bold = true
				case 2:
					label.SetText("电源模块")
					label.TextStyle.Bold = true
				case 3:
					label.SetText("L1相")
					label.TextStyle.Bold = true
				case 4:
					label.SetText("")
					label.TextStyle.Bold = true
				}
				return
			}

			// 数据行（从 row==1 开始）
			r := id.Row - 1
			if r < 0 {
				label.SetText("")
				return
			}

			// 获取最新的校准数据（如果有多条数据，显示最后一条）
			var sourceVoltage, sourceCurrent, sourcePower, sourcePF float64
			//	var sourceReactivePower, sourceActiveEnergy, sourceReactiveEnergy float64
			var pduVoltage, pduCurrent, pduPower, pduPF float64
			var pduReactivePower, pduActiveEnergy, pduReactiveEnergy float64
			if len(lastCalibrationData) > 0 {
				lastData := lastCalibrationData[len(lastCalibrationData)-1]
				sourceVoltage = lastData.SourceVoltage / 10.0 // 从10倍缩放转换为实际值
				sourceCurrent = lastData.SourceCurrent / 10.0
				sourcePower = lastData.SourcePower / 10.0
				sourcePF = lastData.SourcePF / 1000.0
				// sourceReactivePower = lastData.SourceReactivePower
				// sourceActiveEnergy = lastData.SourceActiveEnergy
				// sourceReactiveEnergy = lastData.SourceReactiveEnergy
				// 电源模块数据显示要求：电压/电流/功率因数按比例缩放后显示
				pduVoltage = lastData.PDUVoltage / 10.0
				pduCurrent = lastData.PDUCurrent / 10.0
				pduPower = lastData.PDUPower / 10.0
				pduPF = lastData.PDUPF / 1000.0
				pduReactivePower = lastData.PDUReactivePower
				pduActiveEnergy = lastData.PDUActiveEnergy
				pduReactiveEnergy = lastData.PDUReactiveEnergy
			}

			switch id.Col {
			case 0:
				// 标准源项名列
				if r < len(standardItems) {
					label.SetText(standardItems[r])
				} else {
					label.SetText("")
				}
			case 1:
				// 标准源 L1 相数据列
				if r < len(standardItems) {
					switch r {
					case 0: // 电压
						label.SetText(fmt.Sprintf("%.2f", sourceVoltage))
					case 1: // 电流
						label.SetText(fmt.Sprintf("%.2f", sourceCurrent))
					case 2: // 功率因数
						label.SetText(fmt.Sprintf("%.3f", sourcePF))
					case 3: // 有功功率
						label.SetText(fmt.Sprintf("%.2f", sourcePower))
					}
				} else {
					label.SetText("")
				}
			case 2:
				// 电源模块项名列
				if r < len(powerModuleItems) {
					label.SetText(powerModuleItems[r])
				} else {
					label.SetText("")
				}
			case 3:
				// 电源模块 L1 相数据列
				if r < len(powerModuleItems) {
					switch r {
					case 0: // 电压
						label.SetText(fmt.Sprintf("%.2f", pduVoltage))
					case 1: // 电流
						label.SetText(fmt.Sprintf("%.2f", pduCurrent))
					case 2: // 功率因数
						label.SetText(fmt.Sprintf("%.3f", pduPF))
					case 3: // 有功功率
						label.SetText(fmt.Sprintf("%.2f", pduPower))
					case 4: // 无功功率
						label.SetText(fmt.Sprintf("%.2f", pduReactivePower))
					case 5: // 有功电能
						label.SetText(fmt.Sprintf("%.4f", pduActiveEnergy))
					case 6: // 无功电能
						label.SetText(fmt.Sprintf("%.4f", pduReactiveEnergy))
					}
				} else {
					label.SetText("")
				}
			case 4:
				// 预留列
				label.SetText("")
			}
		},
	)

	// 设置列宽（按需调整）
	calibrationTable.SetColumnWidth(0, 150) // 标准源项名
	calibrationTable.SetColumnWidth(1, 120) // 标准源 L1 数据
	calibrationTable.SetColumnWidth(2, 150) // 电源模块项名
	calibrationTable.SetColumnWidth(3, 120) // 电源模块 L1 数据
	calibrationTable.SetColumnWidth(4, 100) // 预留

	return container.NewBorder(nil, nil, nil, nil, calibrationTable)
}

// UpdateCalibrationTable 更新表格数据并刷新显示
func UpdateCalibrationTable(data []types.CalibrationData) {
	fyne.Do(func() {
		lastCalibrationData = data
		if calibrationTable != nil {
			calibrationTable.Refresh()
		}
	})
}

// StartLiveRefresh 开始实时刷新标准源和PDU数据
func StartLiveRefresh(interval time.Duration) {
	if liveStopChan != nil {
		return // already running
	}
	liveRefreshTick = interval
	liveStopChan = make(chan struct{})
	liveTicker = time.NewTicker(liveRefreshTick)

	go func() {
		for {
			select {
			case <-liveTicker.C:
				// 读取标准电源数据
				src, err1 := power_source.ReadData()
				pduData, err2 := pdu.ReadData()
				var d types.CalibrationData
				if err1 == nil {
					// 按照 types 中的约定：电压/电流/功率放大10倍，PF放大1000倍
					d.SourceVoltage = src.Voltage * 10.0
					d.SourceCurrent = src.Current * 10.0
					d.SourcePower = src.Power * 10.0
					d.SourcePF = src.PF * 1000.0
				}
				if err2 == nil {
					d.PDUVoltage = pduData.Voltage
					d.PDUCurrent = pduData.Current
					d.PDUPower = pduData.Power
					d.PDUPF = pduData.PF
				}
				// 单元素更新，覆盖显示最新一条
				fyne.Do(func() {
					lastCalibrationData = []types.CalibrationData{d}
					if calibrationTable != nil {
						calibrationTable.Refresh()
					}
				})
			case <-liveStopChan:
				liveTicker.Stop()
				liveTicker = nil
				close(liveStopChan)
				liveStopChan = nil
				return
			}
		}
	}()
}

// StopLiveRefresh 停止实时刷新
func StopLiveRefresh() {
	if liveStopChan == nil {
		return
	}
	liveStopChan <- struct{}{}
}
