/*
 * @Author: zzh
 * @Date: 2025-10-23 08:44:47
 * @LastEditors: zzh
 * @LastEditTime: 2025-12-11
 * @Description: 公用的类型定义 - 校准工具简化版
 */
package types

type CheckResult int

const (
	NONE CheckResult = 0 // 无检测结果(异常终止或还未开始检测)
	PASS CheckResult = 1 // 检测通过
	NG   CheckResult = 2 // 检测不通过
)

// 校准数据结构
type CalibrationData struct {
	TestCurrent float64 // 测试电流 (A)

	// 标准电源数据
	SourceVoltage float64 // 电压 (V)
	SourceCurrent float64 // 电流 (A)
	SourcePower   float64 // 功率 (W)
	SourcePF      float64 // 功率因数

	// PDU读取数据
	PDUVoltage float64 // 电压 (V)
	PDUCurrent float64 // 电流 (A)
	PDUPower   float64 // 功率 (W)
	PDUPF      float64 // 功率因数

	// 偏差
	VoltageDiff float64 // 电压偏差
	CurrentDiff float64 // 电流偏差
	PowerDiff   float64 // 功率偏差
	PFDiff      float64 // 功率因数偏差

	// 标准电源计算数据
	SourceReactivePower  float64 // 无功功率 (VAR)
	SourceActiveEnergy   float64 // 有功电能 (Wh)
	SourceReactiveEnergy float64 // 无功电能 (VARh)

	// PDU 计算数据
	PDUReactivePower  float64 // 无功功率 (VAR)
	PDUActiveEnergy   float64 // 有功电能 (Wh)
	PDUReactiveEnergy float64 // 无功电能 (VARh)

	// 结果
	Result CheckResult
}
