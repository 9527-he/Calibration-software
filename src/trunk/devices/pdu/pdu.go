/*
 * @Author: zzh
 * @Date: 2025-11-11 12:14:04
 * @LastEditors:
 * @LastEditTime: 2025-12-10 17:14:28
 * @Description: PDU设备通信模块 - 使用串口命令替代 Modbus
 */
package pdu

import (
	"fmt"
	"main/config"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
)

type PDUData struct {
	Voltage float64 // 电压 (V)
	Current float64 // 电流 (A)
	Power   float64 // 功率 (W)
	PF      float64 // 功率因数
}

var (
	sp *serial.Port = nil
)

// 打开PDU串口连接
func Open(com string) error {
	if sp != nil {
		return fmt.Errorf("device already open")
	}
	baud := 19200
	if config.Serial.PDUBaud > 0 {
		baud = config.Serial.PDUBaud
	}
	rt := time.Second
	if config.Serial.ReadTimeoutMs > 0 {
		rt = time.Millisecond * time.Duration(config.Serial.ReadTimeoutMs)
	}
	cfg := &serial.Config{Name: `\\.\` + com, Baud: baud, ReadTimeout: rt}
	p, err := serial.OpenPort(cfg)
	if err != nil {
		return fmt.Errorf("打开PDU串口失败:\n %w", err)
	}
	sp = p
	return nil
}

// 关闭连接
func Close() {
	if sp != nil {
		_ = sp.Close()
		sp = nil
	}
}

// SendCommandByName sends a named raw byte command from configuration and
// returns any bytes read from the device (may be empty).
func SendCommandByName(name string) ([]byte, error) {
	if sp == nil {
		return nil, fmt.Errorf("device not open")
	}
	if config.Serial.PDUCommands == nil {
		return nil, fmt.Errorf("没有找到PDU命令配置")
	}
	seq, ok := config.Serial.PDUCommands[name]
	if !ok {
		return nil, fmt.Errorf("未配置的PDU命令: %s", name)
	}

	// parse sequence of hex/decimal strings to bytes
	buf := make([]byte, 0, len(seq))
	for _, s := range seq {
		s = strings.TrimSpace(s)
		v, err := strconv.ParseUint(s, 0, 8)
		if err != nil {
			vv, err2 := strconv.ParseUint(s, 10, 8)
			if err2 != nil {
				return nil, fmt.Errorf("无法解析PDU命令字节 '%s': %v/%v", s, err, err2)
			}
			v = vv
		}
		buf = append(buf, byte(v))
	}

	fmt.Printf("PDU发送(%s) bytes: % X\n", name, buf)
	if _, err := sp.Write(buf); err != nil {
		return nil, fmt.Errorf("写入PDU命令失败: %w", err)
	}

	// 读取响应，循环读取直到超时或收到完整数据
	rt := time.Second
	if config.Serial.ReadTimeoutMs > 0 {
		rt = time.Millisecond * time.Duration(config.Serial.ReadTimeoutMs)
	}
	deadline := time.Now().Add(rt)
	respBuf := make([]byte, 0, 256)
	tmpBuf := make([]byte, 256)

	for time.Now().Before(deadline) {
		n, _ := sp.Read(tmpBuf)
		if n > 0 {
			respBuf = append(respBuf, tmpBuf[:n]...)
			// 根据命令类型判断是否收到完整数据
			if name == "calibrate" && len(respBuf) >= 15 {
				break
			}
			if name == "read" && len(respBuf) >= 54 {
				break
			}
			if name == "clear_energy" && len(respBuf) >= 8 {
				break
			}
		}
		// 短暂休眠，避免过度占用CPU
		time.Sleep(10 * time.Millisecond)
	}

	if len(respBuf) > 0 {
		fmt.Printf("PDU响应(%s) bytes: % X\n", name, respBuf)
		return respBuf, nil
	}

	fmt.Printf("PDU响应(%s): 无响应\n", name)
	return nil, nil
}

// ReadData queries the PDU for measured values. It will use a configured
// PDUCommands["read"] if present, otherwise it sends the ASCII "MEAS?\n".
// The function reads up to the configured ReadTimeout and attempts to parse
// an ASCII line like "V:220.000,I:6.000,P:1320.000,PF:0.980".
func ReadData() (PDUData, error) {
	var data PDUData
	if sp == nil {
		return data, fmt.Errorf("device not open")
	}

	// Choose command
	if config.Serial.PDUCommands != nil {
		if _, ok := config.Serial.PDUCommands["read"]; ok {
			resp, err := SendCommandByName("read")
			if err != nil {
				return data, err
			}
			if len(resp) == 0 {
				return data, fmt.Errorf("空响应")
			}
			// parse proprietary binary response per spec
			// verify header
			if len(resp) < 12 {
				return data, fmt.Errorf("PDU响应长度过短: %d", len(resp))
			}
			if resp[0] != 0x96 || resp[1] != 0xAB {
				return data, fmt.Errorf("PDU响应头不匹配: % X", resp[:4])
			}
			// verify xor checksum
			chk := byte(0)
			for i := 0; i < len(resp)-1; i++ {
				chk ^= resp[i]
			}
			if chk != resp[len(resp)-1] {
				return data, fmt.Errorf("PDU响应校验失败: got %02X want %02X", resp[len(resp)-1], chk)
			}

			// parse fields (big-endian 2-byte words) starting at byte index 7 (0-based)
			// positions: [7-8]=V1, [9-10]=V2, [11-12]=V3, [13-14]=I1, [15-16]=I2, [17-18]=I3,
			// [19-20]=P1, [21-22]=P2, [23-24]=P3, [25-26]=PF1, [27-28]=PF2, [29-30]=PF3
			if len(resp) < 31 {
				return data, fmt.Errorf("PDU响应长度不足以解析电参数: %d", len(resp))
			}
			be := func(h, l byte) uint16 { return uint16(h)<<8 | uint16(l) }
			idx := 6
			v1 := be(resp[idx], resp[idx+1])
			idx += 2
			_ = be(resp[idx], resp[idx+1]) // v2
			idx += 2
			_ = be(resp[idx], resp[idx+1]) // v3
			idx += 2
			i1 := be(resp[idx], resp[idx+1])
			idx += 2
			_ = be(resp[idx], resp[idx+1]) // i2
			idx += 2
			_ = be(resp[idx], resp[idx+1]) // i3
			idx += 2
			_ = be(resp[idx], resp[idx+1])
			idx += 2
			_ = be(resp[idx], resp[idx+1]) // p2
			idx += 2
			_ = be(resp[idx], resp[idx+1]) // p3
			idx += 2
			idx += 18
			pf1 := be(resp[idx], resp[idx+1])
			idx += 2
			_ = be(resp[idx], resp[idx+1]) // pf2
			idx += 2
			_ = be(resp[idx], resp[idx+1]) // pf3
			idx += 2

			// 打印一相原始值和转换后的值
			fmt.Printf("PDU一相数据: v1=%d, i1=%d, pf1=%d \n", v1, i1, pf1)

			// aggregate into PDUData: Voltage = average phase voltage, Current = sum of phase currents,
			// Power = sum of phase active powers, PF = average PF
			data.Voltage = float64(v1)
			data.Current = float64(i1) / 10.0
			data.Power = float64(v1) * float64(i1) / 10.0 / 10.0
			data.PF = 1000.0
			fmt.Printf("计算功率: v1=%d, i1=%d, data.Power=%.2f, data.PF=%.3f\n", v1, i1, data.Power, data.PF)
			return data, nil
		}
	}

	return data, fmt.Errorf("未配置PDU读取命令")
}

// Calibrate writes calibration values to the PDU. If specific PDUCommands
// are configured , they will be used; otherwise, an error is returned.
func Calibrate(voltage float64, current float64, power float64) error {
	if sp == nil {
		return fmt.Errorf("device not open")
	}
	// Use configured "calibrate" command
	if config.Serial.PDUCommands != nil {
		if _, ok := config.Serial.PDUCommands["calibrate"]; ok {
			resp, err := SendCommandByName("calibrate")
			if err != nil {
				return fmt.Errorf("发送校准命令失败: %w", err)
			}
			// verify response starts with 0xD1 and ends with 0x7D (as provided)
			if len(resp) == 0 {
				return fmt.Errorf("校准响应为空，未收到任何数据")
			}
			if len(resp) < 15 {
				return fmt.Errorf("校准响应长度不足:\n收到%d字节(期望15字节), 数据: % X", len(resp), resp)
			}
			if resp[0] != 0xD1 {
				return fmt.Errorf("校准响应头字节错误:\n期望0xD1, 实际0x%02X, 完整数据: % X", resp[0], resp)
			}
			// 验证异或校验:最后一个字节应等于前面所有字节的异或值
			chk := byte(0)
			for i := 0; i < len(resp)-1; i++ {
				chk ^= resp[i]
			}
			if chk != resp[len(resp)-1] {
				return fmt.Errorf("校准响应校验失败: 计算值0x%02X,\n实际值0x%02X, 完整数据: % X", chk, resp[len(resp)-1], resp)
			}
			fmt.Printf("PDU校准成功,响应: % X\n", resp)
			return nil
		}
	}
	return fmt.Errorf("未配置PDU校准命令")
}

// ClearEnergy clears PDU active energy. Uses configured "clear_energy" command
// if available, otherwise returns an error.
func ClearEnergy() error {
	if sp == nil {
		return fmt.Errorf("device not open")
	}
	if config.Serial.PDUCommands != nil {
		if _, ok := config.Serial.PDUCommands["clear_energy"]; ok {
			_, err := SendCommandByName("clear_energy")
			return err
		}
	}

	return fmt.Errorf("未配置PDU电能清零命令")
}

// ComputeReactivePowerFromPF 通过电压 V (V)，电流 I (A) 和功率因数 PF 计算无功功率 Q (VAR)
// 公式：S = V * I，P = S * PF，Q = sqrt(max(0, S^2 - P^2))
func ComputeReactivePowerFromPF(V, I, PF float64) float64 {
	if PF > 1.0 {
		PF = 1.0
	}
	if PF < -1.0 {
		PF = -1.0
	}
	S := V * I
	P := S * PF
	q2 := S*S - P*P
	if q2 <= 0 {
		return 0
	}
	return math.Sqrt(q2)
}

// ComputeReactivePowerFromP 通过电压 V (V)、电流 I (A) 和有功功率 P (W) 计算无功功率 Q (VAR)
// 公式：S = V * I，Q = sqrt(max(0, S^2 - P^2))
func ComputeReactivePowerFromP(V, I, P float64) float64 {
	S := V * I
	q2 := S*S - P*P
	if q2 <= 0 {
		return 0
	}
	return math.Sqrt(q2)
}

// ComputeEnergyWh 计算有功电能 Wh（瓦时）
// P: 有功功率 (W)
// durationSec: 持续时间，单位秒
func ComputeEnergyWh(P, durationSec float64) float64 {
	return P * durationSec / 3600.0
}

// ComputeReactiveEnergyVARh 计算无功电能 VARh（无功伏安时）
// Q: 无功功率 (VAR)
// durationSec: 持续时间，单位秒
func ComputeReactiveEnergyVARh(Q, durationSec float64) float64 {
	return Q * durationSec / 3600.0
}
