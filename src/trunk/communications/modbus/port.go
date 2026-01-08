/*
 * @Author: zzh
 * @Date: 2025-11-04 09:35:13
 * @LastEditors: zzh
 * @LastEditTime: 2025-11-05 09:26:19
 * @Description: 端口管理
 */
package modbus

import (
	"fmt"
	"log"
	"main/config"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goburrow/modbus"
)

type PortOption func(*portConfig)

type portConfig struct {
	baud   int
	data   int
	stop   int
	parity string
}

func WithBaud(baud int) PortOption {
	return func(c *portConfig) { c.baud = baud }
}

func WithParity(parity string) PortOption {
	return func(c *portConfig) { c.parity = parity }
}

type Port struct {
	handler    *modbus.RTUClientHandler
	client     modbus.Client
	users      atomic.Int32 // 引用计数
	com        string       // 串口号，仅日志用
	closeMutex sync.Mutex   // 保护关闭逻辑的互斥锁
	opMutex    sync.Mutex   // 保护端口操作的互斥锁
	closed     bool
}

// Open 第一次调用时打开硬件 后续仅引用计数 +1
func (port *Port) Open(com string, opts ...PortOption) error {
	port.closeMutex.Lock()
	defer port.closeMutex.Unlock()

	if port.handler != nil { // 已打开过，直接增加引用计数
		port.users.Add(1)
		return nil
	}

	cfg := &portConfig{baud: 9600, data: 8, stop: 1, parity: "N"}
	for _, opt := range opts {
		opt(cfg)
	}

	port.handler = modbus.NewRTUClientHandler(`\\.\` + com)
	port.com = com
	port.handler.BaudRate = cfg.baud
	port.handler.DataBits = cfg.data
	port.handler.StopBits = cfg.stop
	port.handler.Parity = cfg.parity
	port.handler.Timeout = 800 * time.Millisecond

	// 调试日志配置
	if config.Debug.Modbus == 1 {
		port.handler.Logger = log.New(os.Stdout, com+" ", log.Ltime|log.Lmicroseconds)
	}

	// 打开端口
	if err := port.handler.Connect(); err != nil {
		port.handler = nil
		return fmt.Errorf("open %s: %w", com, err)
	}
	
	port.client = modbus.NewClient(port.handler)
	port.users.Store(1) // 初始引用计数1
	port.closed = false
	return nil
}

// Close 引用计数-1 到 0 才真正关闭端口
func (port *Port) Close() error {
	port.closeMutex.Lock()
	defer port.closeMutex.Unlock()

	currentUsers := port.users.Load()
	if currentUsers <= 0 {
		return nil
	}

	newUsers := port.users.Add(-1)
	if newUsers == 0 {
		if port.closed {
			return nil
		}
		port.closed = true
		if err := port.handler.Close(); err != nil {
			return fmt.Errorf("close port %s: %w", port.com, err)
		}
		port.handler = nil
		port.client = nil
	}
	return nil
}

func (port *Port) ReadHoldingRegisters(id byte, address, quantity uint16) (results []byte, err error) {
	port.opMutex.Lock()
	defer port.opMutex.Unlock()
	port.handler.SlaveId = id
	return port.client.ReadHoldingRegisters(address, quantity)
}

func (port *Port) ReadInputRegisters(id byte, address, quantity uint16) (results []byte, err error) {
	port.opMutex.Lock()
	defer port.opMutex.Unlock()
	port.handler.SlaveId = id
	return port.client.ReadInputRegisters(address, quantity)
}

func (port *Port) WriteSingleRegister(id byte, address, value uint16) (results []byte, err error) {
	port.opMutex.Lock()
	defer port.opMutex.Unlock()
	port.handler.SlaveId = id
	return port.client.WriteSingleRegister(address, value)
}

func (port *Port) WriteMultipleRegisters(id byte, address, quantity uint16, value []byte) (results []byte, err error) {
	port.opMutex.Lock()
	defer port.opMutex.Unlock()
	port.handler.SlaveId = id
	return port.client.WriteMultipleRegisters(address, quantity, value)
}

func (port *Port) ReadCoils(id byte, address, quantity uint16) (results []byte, err error) {
	port.opMutex.Lock()
	defer port.opMutex.Unlock()
	port.handler.SlaveId = id
	return port.client.ReadCoils(address, quantity)
}

func (port *Port) ReadDiscreteInputs(id byte, address, quantity uint16) (results []byte, err error) {
	port.opMutex.Lock()
	defer port.opMutex.Unlock()
	port.handler.SlaveId = id
	return port.client.ReadDiscreteInputs(address, quantity)
}

func (port *Port) WriteSingleCoil(id byte, address, value uint16) (results []byte, err error) {
	port.opMutex.Lock()
	defer port.opMutex.Unlock()
	port.handler.SlaveId = id
	return port.client.WriteSingleCoil(address, value)
}

func (port *Port) WriteMultipleCoils(id byte, address, quantity uint16, value []byte) (results []byte, err error) {
	port.opMutex.Lock()
	defer port.opMutex.Unlock()
	port.handler.SlaveId = id
	return port.client.WriteMultipleCoils(address, quantity, value)
}
