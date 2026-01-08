/*
 * @Author: zzh
 * @Date: 2025-12-08
 * @Description: 标准电源通信模块
 */
package power_source

import (
	"bufio"
	"fmt"
	"main/config"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
)

type PowerData struct {
	Voltage float64 // 电压 (V)
	Current float64 // 电流 (A)
	Power   float64 // 功率 (W)
	PF      float64 // 功率因数
}

var (
	sp *serial.Port = nil
)

// 打开标准电源串口连接
func Open(com string) error {
	if sp != nil {
		return fmt.Errorf("device already open")
	}
	baud := 38400
	if config.Serial.SourceBaud > 0 {
		baud = config.Serial.SourceBaud
	}
	rt := time.Second
	if config.Serial.ReadTimeoutMs > 0 {
		rt = time.Millisecond * time.Duration(config.Serial.ReadTimeoutMs)
	}
	cfg := &serial.Config{Name: `\\.\` + com, Baud: baud, ReadTimeout: rt}
	p, err := serial.OpenPort(cfg)
	if err != nil {
		return fmt.Errorf("打开标准电源串口失败:\n %w", err)
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

// 设置标准电源输出
// voltage: 电压 (V)
// current: 电流 (A)
// SetOutput sets the source output. This implementation sends simple ASCII
// commands over the serial port. The exact command format depends on the
// standard source device. The current format used here is a generic one:
//
//	SETV <voltage>\n
//	SETI <current>\n
//
// and expects an "OK" or similar short response.
// Replace commands below to match your equipment's protocol.
func SetOutput(voltage float64, current float64) error {
	if sp == nil {
		return fmt.Errorf("device not open")
	}

	// Set voltage
	vcmd := fmt.Sprintf("SETV %.3f\n", voltage)
	if _, err := sp.Write([]byte(vcmd)); err != nil {
		return fmt.Errorf("写入设置电压命令失败: %w", err)
	}
	// read ack (non-blocking with timeout provided by port config)
	rdr := bufio.NewReader(sp)
	ack, _ := rdr.ReadString('\n')
	ack = strings.TrimSpace(ack)
	if ack == "" {
		// not fatal; some devices don't ack
	}

	// Set current
	icmd := fmt.Sprintf("SETI %.3f\n", current)
	if _, err := sp.Write([]byte(icmd)); err != nil {
		return fmt.Errorf("写入设置电流命令失败: %w", err)
	}
	ack2, _ := rdr.ReadString('\n')
	_ = strings.TrimSpace(ack2)

	fmt.Printf("标准电源设置: %.3fV, %.3fA\n", voltage, current)
	// allow device settle
	time.Sleep(200 * time.Millisecond)
	return nil
}

// SendCommandByName sends a named raw byte command from configuration.
// SendCommandByName sends a named raw byte command from configuration and
// returns any bytes/text read from the device (may be empty).
func SendCommandByName(name string) (resp []byte, err error) {
	if sp == nil {
		return nil, fmt.Errorf("device not open")
	}
	if config.Serial.PowerSourceCommands == nil {
		return nil, fmt.Errorf("没有找到标准电源命令配置")
	}
	seq, ok := config.Serial.PowerSourceCommands[name]
	if !ok {
		return nil, fmt.Errorf("未配置的命令: %s", name)
	}

	// parse sequence of hex/decimal strings to bytes
	buf := make([]byte, 0, len(seq))
	for _, s := range seq {
		s = strings.TrimSpace(s)
		v, err := strconv.ParseUint(s, 0, 8)
		if err != nil {
			// try decimal
			vv, err2 := strconv.ParseUint(s, 10, 8)
			if err2 != nil {
				return nil, fmt.Errorf("无法解析命令字节 '%s': %v/%v", s, err, err2)
			}
			v = vv
		}
		buf = append(buf, byte(v))
	}

	if _, err := sp.Write(buf); err != nil {
		return nil, fmt.Errorf("写入命令失败: %w", err)
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
			// 如果收到的字节数达到预期长度(8字节)，提前返回
			if len(respBuf) >= 8 {
				break
			}
		}
		// 短暂休眠，避免过度占用CPU
		time.Sleep(10 * time.Millisecond)
	}

	if len(respBuf) > 0 {
		fmt.Printf("标准电源响应(%s) bytes: % X\n", name, respBuf)
		return respBuf, nil
	}

	fmt.Printf("标准电源响应(%s): 无响应\n", name)
	return nil, nil
}

// 读取标准电源数据
// ReadData queries the instrument for measured values. The current implementation
// sends the ASCII command "MEAS?\n" and expects a single-line comma-separated
// response such as:
//
//	V:220.000,I:6.000,P:1320.000,PF:0.980\n
//
// Adjust the parsing logic for your device's actual response format.
func ReadData() (PowerData, error) {
	var data PowerData
	if sp == nil {
		return data, fmt.Errorf("device not open")
	}

	// send measurement query
	if _, err := sp.Write([]byte("MEAS?\n")); err != nil {
		return data, fmt.Errorf("写入读取命令失败: %w", err)
	}

	rdr := bufio.NewReader(sp)
	line, err := rdr.ReadString('\n')
	if err != nil {
		return data, fmt.Errorf("读取响应失败: %w", err)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return data, fmt.Errorf("空响应")
	}

	return parseMeasurementLine(line)
}

// parseMeasurementLine parses a single-line measurement response such as
// "V:220.000,I:6.000,P:1320.000,PF:0.980" into PowerData.
func parseMeasurementLine(line string) (PowerData, error) {
	var data PowerData
	parts := strings.Split(line, ",")
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(kv[0]))
		val := strings.TrimSpace(kv[1])
		switch key {
		case "V", "VOLT", "VOLTAGE":
			if f, e := strconv.ParseFloat(val, 64); e == nil {
				data.Voltage = f
			}
		case "I", "CURR", "CURRENT":
			if f, e := strconv.ParseFloat(val, 64); e == nil {
				data.Current = f
			}
		case "P", "POWER":
			if f, e := strconv.ParseFloat(val, 64); e == nil {
				data.Power = f
			}
		case "PF", "PFACT", "POWERFACTOR":
			if f, e := strconv.ParseFloat(val, 64); e == nil {
				data.PF = f
			}
		}
	}
	return data, nil
}

// ConfigureAndVerify sends the named command to configure the power source
func ConfigureAndVerify(name string) (PowerData, error) {
	var last PowerData
	// send command and capture immediate response (if any)
	respBytes, err := SendCommandByName(name)
	if err != nil {
		return last, err
	}

	// 解析响应判断标准源设置是否成功
	if len(respBytes) != 8 {
		return last, fmt.Errorf("\n标准源响应长度异常: %d bytes,\n 期望8 bytes", len(respBytes))
	}

	// 成功响应: 68 08 00 68 80 10 90 16
	successResp := []byte{0x68, 0x08, 0x00, 0x68, 0x80, 0x10, 0x90, 0x16}
	// 失败响应1: 68 08 00 68 80 80 00 16
	failResp1 := []byte{0x68, 0x08, 0x00, 0x68, 0x80, 0x80, 0x00, 0x16}
	// 失败响应2: 68 08 00 68 80 80 90 16
	failResp2 := []byte{0x68, 0x08, 0x00, 0x68, 0x80, 0x80, 0x90, 0x16}

	isSuccess := true
	isFail1 := true
	isFail2 := true
	for i := 0; i < 8; i++ {
		if respBytes[i] != successResp[i] {
			isSuccess = false
		}
		if respBytes[i] != failResp1[i] {
			isFail1 = false
		}
		if respBytes[i] != failResp2[i] {
			isFail2 = false
		}
	}

	if isFail1 || isFail2 {
		return last, fmt.Errorf("标准源设置失败，响应: % X", respBytes)
	}
	if !isSuccess {
		return last, fmt.Errorf("标准源响应未知: % X", respBytes)
	}

	// 根据命令名称填充对应的电参数值
	// 电压/电流/功率放大10倍，功率因数放大1000倍
	switch name {
	case "single_220_6":
		last.Voltage = 2200
		last.Current = 60
		last.Power = 13200
		last.PF = 1000
	case "single_200_3":
		last.Voltage = 2000
		last.Current = 30
		last.Power = 6000
		last.PF = 1000
	case "three_220_6":
		last.Voltage = 2200
		last.Current = 60
		last.Power = 13200
		last.PF = 1000
	case "three_200_3":
		last.Voltage = 2000
		last.Current = 30
		last.Power = 6000
		last.PF = 1000
	}

	return last, nil
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
