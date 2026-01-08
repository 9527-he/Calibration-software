/*
 * @Author: zzh
 * @Date: 2025-09-28 14:16:11
 * @LastEditors:
 * @LastEditTime: 2025-12-11 09:10:33
 * @Description: PDU校准工具核心逻辑
 */
package check

import (
	"fmt"
	"main/config"
	"main/devices/pdu"
	"main/devices/power_source"
	"main/gui"
	"main/gui/customWidget"
	"main/gui/table"
	"main/gui/window"
	"main/types"
	"os"
	"time"

	"github.com/xuri/excelize/v2"
)

// 校准数据结构已移至types包

var calibrationData []types.CalibrationData

// 执行校准流程
func Run(startTime time.Time) error {
	var err error

	sn := window.Info.Sn

	// 1. 打开串口连接
	window.SetMultiFuncButtonText("连接设备中...")
	if err = connectDevices(); err != nil {
		return fmt.Errorf("设备连接失败: %w", err)
	}
	defer disconnectDevices()

	// 2. 设置标准电源为220V, 6A输出并确认
	window.SetMultiFuncButtonText("设置标准电源...")
	sourceData, err := power_source.ConfigureAndVerify("single_220_6")
	if err != nil {
		return fmt.Errorf("设置/确认标准电源失败: %w", err)
	}
	fmt.Println("标准电源6A配置完成")

	// 3. 启动标准电源输出
	window.SetMultiFuncButtonText("启动标准电源输出...")
	_, err = power_source.ConfigureAndVerify("start_L1")
	if err != nil {
		return fmt.Errorf("启动标准电源输出失败: %w", err)
	}
	fmt.Println("标准电源输出已启动")

	// 延时8秒等待电源稳定
	window.SetMultiFuncButtonText("等待电源稳定...")
	time.Sleep(3 * time.Second)
	fmt.Println("电源稳定，继续校准")

	// 4. 校准PDU (将标准电源的值写入PDU)
	window.SetMultiFuncButtonText("校准PDU中...")
	if err = pdu.Calibrate(sourceData.Voltage, sourceData.Current, sourceData.Power); err != nil {
		return fmt.Errorf("PDU校准失败: %w", err)
	}
	fmt.Println("PDU校准完成")
	time.Sleep(5 * time.Second)
	// 5. 第一次测试 (6A负载)
	window.SetMultiFuncButtonText("6A负载测试中...")
	test1 := types.CalibrationData{TestCurrent: 6.0}
	test1.SourceVoltage = sourceData.Voltage
	test1.SourceCurrent = sourceData.Current
	test1.SourcePower = sourceData.Power
	test1.SourcePF = sourceData.PF

	calibrationData = append(calibrationData, test1)
	table.UpdateCalibrationTable(calibrationData)

	pduData, err := pdu.ReadData()
	if err != nil {
		return fmt.Errorf("读取PDU数据失败: %w", err)
	}

	test1.PDUVoltage = pduData.Voltage
	test1.PDUCurrent = pduData.Current
	test1.PDUPower = pduData.Power
	test1.PDUPF = pduData.PF

	test1.VoltageDiff = abs(test1.PDUVoltage - test1.SourceVoltage)
	test1.CurrentDiff = abs(test1.PDUCurrent - test1.SourceCurrent)
	test1.PowerDiff = abs(test1.PDUPower - test1.SourcePower)
	test1.PFDiff = abs(test1.PDUPF - test1.SourcePF)

	calibrationData = append(calibrationData, test1)
	table.UpdateCalibrationTable(calibrationData)

	fmt.Printf("6A测试 - 标准源无功功率: %.2f VAR, 有功电能: %.4f Wh, 无功电能: %.4f VARh\n",
		test1.SourceReactivePower, test1.SourceActiveEnergy, test1.SourceReactiveEnergy)
	fmt.Printf("6A测试 - PDU无功功率: %.2f VAR, 有功电能: %.4f Wh, 无功电能: %.4f VARh\n",
		test1.PDUReactivePower, test1.PDUActiveEnergy, test1.PDUReactiveEnergy)

	fmt.Printf("6A测试 - PDU: V=%.2fV, I=%.2fA, P=%.2fW, PF=%.3f\n",
		test1.PDUVoltage, test1.PDUCurrent, test1.PDUPower, test1.PDUPF)
	fmt.Printf("6A测试 - 偏差: ΔV=%.2fV, ΔI=%.3fA, ΔP=%.2fW, ΔPF=%.3f\n",
		test1.VoltageDiff, test1.CurrentDiff, test1.PowerDiff, test1.PFDiff)

	if checkThreshold(test1) {
		test1.Result = types.PASS
		fmt.Println("6A测试: PASS")
	} else {
		test1.Result = types.NG
		fmt.Println("6A测试: NG")
	}

	// 6. 设置标准电源为200V, 3A输出并确认
	window.SetMultiFuncButtonText("设置3A负载...")
	sourceData, err = power_source.ConfigureAndVerify("single_200_3")
	if err != nil {
		return fmt.Errorf("设置/确认标准电源失败: %w", err)
	}
	fmt.Println("标准电源3A配置完成")

	// 7. 启动标准电源输出
	window.SetMultiFuncButtonText("启动标准电源输出...")
	_, err = power_source.ConfigureAndVerify("start_L1")
	if err != nil {
		return fmt.Errorf("启动标准电源输出失败: %w", err)
	}
	fmt.Println("标准电源输出已启动")

	// 延时5秒等待电源稳定
	window.SetMultiFuncButtonText("等待电源稳定...")
	time.Sleep(3 * time.Second)
	fmt.Println("电源稳定，继续校准")

	// 8. 第二次测试 (3A负载)
	window.SetMultiFuncButtonText("3A负载测试中...")

	test2 := types.CalibrationData{TestCurrent: 3.0}
	test2.SourceVoltage = sourceData.Voltage
	test2.SourceCurrent = sourceData.Current
	test2.SourcePower = sourceData.Power
	test2.SourcePF = sourceData.PF

	fmt.Printf("3A测试 - 标准源: V=%.2fV, I=%.2fA, P=%.2fW, PF=%.3f\n",
		test2.SourceVoltage, test2.SourceCurrent, test2.SourcePower, test2.SourcePF)

	pduData, err = pdu.ReadData()
	if err != nil {
		return fmt.Errorf("读取PDU数据失败: %w", err)
	}
	test2.PDUVoltage = pduData.Voltage
	test2.PDUCurrent = pduData.Current
	test2.PDUPower = pduData.Power
	test2.PDUPF = pduData.PF

	test2.VoltageDiff = abs(test2.PDUVoltage - test2.SourceVoltage)
	test2.CurrentDiff = abs(test2.PDUCurrent - test2.SourceCurrent)
	test2.PowerDiff = abs(test2.PDUPower - test2.SourcePower)
	test2.PFDiff = abs(test2.PDUPF - test2.SourcePF)

	// 计算无功功率 (VAR)
	sourceVoltageV2 := test2.SourceVoltage / 10.0
	sourceCurrentA2 := test2.SourceCurrent / 10.0
	sourcePowerW2 := test2.SourcePower / 10.0
	sourcePF2 := test2.SourcePF / 1000.0
	test2.SourceReactivePower = power_source.ComputeReactivePowerFromPF(sourceVoltageV2, sourceCurrentA2, sourcePF2)

	pduVoltageV2 := test2.PDUVoltage / 10.0
	pduCurrentA2 := test2.PDUCurrent / 10.0
	pduPowerW2 := test2.PDUPower / 10.0
	pduPF2 := test2.PDUPF / 1000.0
	test2.PDUReactivePower = pdu.ComputeReactivePowerFromPF(pduVoltageV2, pduCurrentA2, pduPF2)

	// 计算有功电能和无功电能（假设测试持续10秒）
	testDurationSec2 := 10.0
	test2.SourceActiveEnergy = power_source.ComputeEnergyWh(sourcePowerW2, testDurationSec2)
	test2.SourceReactiveEnergy = power_source.ComputeReactiveEnergyVARh(test2.SourceReactivePower, testDurationSec2)
	test2.PDUActiveEnergy = pdu.ComputeEnergyWh(pduPowerW2, testDurationSec2)
	test2.PDUReactiveEnergy = pdu.ComputeReactiveEnergyVARh(test2.PDUReactivePower, testDurationSec2)

	//更新表格数据
	calibrationData = append(calibrationData, test2)
	table.UpdateCalibrationTable(calibrationData)
	time.Sleep(3 * time.Second)

	fmt.Printf("3A测试 - 标准源无功功率: %.2f VAR, 有功电能: %.4f Wh, 无功电能: %.4f VARh\n",
		test2.SourceReactivePower, test2.SourceActiveEnergy, test2.SourceReactiveEnergy)
	fmt.Printf("3A测试 - PDU无功功率: %.2f VAR, 有功电能: %.4f Wh, 无功电能: %.4f VARh\n",
		test2.PDUReactivePower, test2.PDUActiveEnergy, test2.PDUReactiveEnergy)

	fmt.Printf("3A测试 - PDU: V=%.2fV, I=%.2fA, P=%.2fW, PF=%.3f\n",
		test2.PDUVoltage, test2.PDUCurrent, test2.PDUPower, test2.PDUPF)
	fmt.Printf("3A测试 - 偏差: ΔV=%.2fV, ΔI=%.3fA, ΔP=%.2fW, ΔPF=%.3f\n",
		test2.VoltageDiff, test2.CurrentDiff, test2.PowerDiff, test2.PFDiff)

	if checkThreshold(test2) {
		test2.Result = types.PASS
		fmt.Println("3A测试: PASS")
	} else {
		test2.Result = types.NG
		fmt.Println("3A测试: NG")
	}

	// 9. 判断总体结果
	finalResult := types.PASS
	if test1.Result == types.NG || test2.Result == types.NG {
		finalResult = types.NG
		customWidget.ShowDialog("测试结果", "校准测试失败 (NG)", "no.png", gui.Window)
		return fmt.Errorf("校准测试未通过")
	}

	// 10. 清零PDU有功电能
	window.SetMultiFuncButtonText("清零PDU有功电能...")
	if err = pdu.ClearEnergy(); err != nil {
		return fmt.Errorf("清零PDU有功电能失败: %w", err)
	}
	fmt.Println("PDU有功电能已清零")
	customWidget.ShowDialog("测试结果", "校准测试通过 (PASS)", "yes.png", gui.Window)
	fmt.Printf("校准流程完成, 最终结果: %v\n", finalResult)

	// 校准完成，发送命令关闭标准电源输出
	window.SetMultiFuncButtonText("关闭标准电源输出...")
	_, err = power_source.ConfigureAndVerify("three_stop")
	if err != nil {
		fmt.Printf("警告：关闭标准电源失败: %v\n", err)
	} else {
		fmt.Println("标准电源已关闭")
	}

	// 有余压,等待3秒确保标准源电压放空,归零
	time.Sleep(3 * time.Second)

	// 追加一行全零数据作为结束标志
	make_zero := types.CalibrationData{TestCurrent: 0}
	make_zero.SourceVoltage = 0
	make_zero.SourceCurrent = 0
	make_zero.SourcePower = 0
	make_zero.SourcePF = 0
	calibrationData = append(calibrationData, make_zero)
	table.UpdateCalibrationTable(calibrationData)

	// 写入校准数据到XLSX
	if err := writeToExcel(sn, startTime, test1, test2); err != nil {
		fmt.Printf("写入XLSX失败: %v\n", err)
	}

	return nil
}

// 连接设备
func connectDevices() error {
	// 连接标准电源
	if err := power_source.Open(window.Info.SourceCOM); err != nil {
		return fmt.Errorf("标准电源连接失败: %w", err)
	}

	// 连接PDU
	if err := pdu.Open(window.Info.PDUCOM); err != nil {
		power_source.Close()
		return fmt.Errorf("PDU连接失败: %w", err)
	}

	return nil
}

// 断开设备连接
func disconnectDevices() {
	pdu.Close()
	power_source.Close()
}

// 检查偏差是否在阈值范围内
func checkThreshold(data types.CalibrationData) bool {
	return data.VoltageDiff <= config.Thresholds.VMaxDiff &&
		data.CurrentDiff <= config.Thresholds.IMaxDiff &&
		data.PowerDiff <= config.Thresholds.PMaxDiff &&
		data.PFDiff <= config.Thresholds.PfMaxDiff
}

// 绝对值
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// 写入校准数据到XLSX文件
func writeToExcel(sn string, startTime time.Time, test6A, test3A types.CalibrationData) error {
	var f *excelize.File
	var err error

	fileName := "02120701校准数据.xlsx"

	const maxSize = 1 * 1024 * 1024 // 10MB
	// 如果基础文件存在且已超过大小限制，则寻找第一个不存在的序号文件名
	if info, err := os.Stat(fileName); err == nil && info.Size() >= maxSize {
		for i := 1; ; i++ {
			candidate := fmt.Sprintf("02120701校准数据_%d.xlsx", i)
			if _, err := os.Stat(candidate); os.IsNotExist(err) {
				fileName = candidate
				break
			}
		}
	}

	if _, err = os.Stat(fileName); os.IsNotExist(err) {
		f = excelize.NewFile()
	} else {
		f, err = excelize.OpenFile(fileName)
		if err != nil {
			return err
		}
	}
	defer f.Close()

	// 工作表名称：使用固定工作表名以持续追加数据，避免每次重启创建新表
	sheetName := "校准数据"

	// 检查工作表是否存在
	index, err := f.GetSheetIndex(sheetName)
	if err != nil || index == -1 {
		// 不存在，创建新工作表
		index, err = f.NewSheet(sheetName)
		if err != nil {
			return err
		}
		// 设置活动工作表
		f.SetActiveSheet(index)
		// 写入标题行
		headers := []string{
			"SN码", "校准时间",
			"标准源电压(V)", "标准源电流(A)", "标准源功率因数", "标准源有功功率(W)",
			"PDU 6A测试电压(V)", "PDU 6A测试电流(A)", "PDU 6A测试功率因数", "PDU 6A测试有功功率(W)",
			"PDU 3A测试电压(V)", "PDU 3A测试电流(A)", "PDU 3A测试功率因数", "PDU 3A测试有功功率(W)",
		}
		for i, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheetName, cell, header)
		}
	}

	// 获取当前工作表的行数
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return err
	}

	// 下一行号
	nextRow := len(rows) + 1

	// 准备数据
	data := []interface{}{
		sn,
		time.Now().Format("2006-01-02 15:04:05"), // 实际校准时间
		test6A.SourceVoltage,
		test6A.SourceCurrent,
		test6A.SourcePF,
		test6A.SourcePower,
		test6A.PDUVoltage,
		test6A.PDUCurrent,
		test6A.PDUPF,
		test6A.PDUPower,
		test3A.PDUVoltage,
		test3A.PDUCurrent,
		test3A.PDUPF,
		test3A.PDUPower,
	}

	// 写入数据行
	for i, value := range data {
		cell, _ := excelize.CoordinatesToCellName(i+1, nextRow)
		f.SetCellValue(sheetName, cell, value)
	}

	// 保存文件
	return f.SaveAs(fileName)
}
