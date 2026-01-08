/*
 * @Author: zzh
 * @Date: 2025-09-17 09:55:07
 * @LastEditors: zzh
 * @LastEditTime: 2025-12-08
 * @Description: 配置管理模块 - 校准工具简化版
 */
package config

import (
	"fmt"
	"main/terminal"
	"os"

	"gopkg.in/yaml.v3"
)

// 阈值配置
type ThresholdsConfig struct {
	VMaxDiff  float64 `yaml:"VMaxDiff"`  // 电压最大偏差 (V)
	IMaxDiff  float64 `yaml:"IMaxDiff"`  // 电流最大偏差 (A)
	PMaxDiff  float64 `yaml:"PMaxDiff"`  // 功率最大偏差 (W)
	PfMaxDiff float64 `yaml:"PfMaxDiff"` // 功率因数最大偏差
}

// 串口配置
type SerialConfig struct {
	SourceCOM string `yaml:"SourceCOM"` // 标准电源串口
	PDUCOM    string `yaml:"PDUCOM"`    // PDU串口
	// PowerSourceCommands: map of named commands, each is a list of hex strings
	PowerSourceCommands map[string][]string `yaml:"PowerSourceCommands"`
	// PDUCommands: map of named commands for the PDU device
	PDUCommands map[string][]string `yaml:"PDUCommands"`
	// Serial params for the standard source port
	SourceBaud       int     `yaml:"SourceBaud"`
	PDUBaud          int     `yaml:"PDUBaud"`
	Parity           string  `yaml:"Parity"`
	ReadTimeoutMs    int     `yaml:"ReadTimeoutMs"`
	VerifyVThreshold float64 `yaml:"VerifyVThreshold"` // voltage verify tolerance (V)
	VerifyIThreshold float64 `yaml:"VerifyIThreshold"` // current verify tolerance (A)
}

// 调试配置
type DebugConfig struct {
	Modbus     byte `yaml:"Modbus"`     // Modbus调试开关
	BreakPoint byte `yaml:"BreakPoint"` // 断点调试开关
}

var (
	Thresholds ThresholdsConfig
	Serial     SerialConfig
	Debug      DebugConfig
)

// 解析配置文件
func parseConfigFile(filename string, out any) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, out)
	if err != nil {
		return err
	}
	return nil
}

// 校验配置
func check() error {
	if Serial.SourceCOM == "" || Serial.PDUCOM == "" {
		return fmt.Errorf("串口配置异常")
	}
	if Thresholds.VMaxDiff <= 0 || Thresholds.IMaxDiff <= 0 {
		return fmt.Errorf("阈值配置异常")
	}
	return nil
}

// 从文件加载配置
func Load() error {
	var err error

	err = parseConfigFile("configs/thresholds.yaml", &Thresholds)
	if err != nil {
		goto err
	}

	err = parseConfigFile("configs/serial.yaml", &Serial)
	if err != nil {
		goto err
	}

	err = parseConfigFile("configs/debug.yaml", &Debug)
	if err != nil {
		goto err
	}

	err = check()
	if err != nil {
		goto err
	}

	return nil

err:
	terminal.WaitExit(err.Error())
	return err
}
