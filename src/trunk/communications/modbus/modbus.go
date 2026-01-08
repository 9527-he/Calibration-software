/*
 * @Author: zzh
 * @Date: 2025-10-29 16:56:56
 * @LastEditors: zzh
 * @LastEditTime: 2025-11-05 10:19:54
 * @Description: Modbus设备管理优化版，修复并发与锁逻辑问题
 */
package modbus

import (
	"sync"
	"sync/atomic"
)

var (
	portMutex sync.RWMutex
	portMap   = make(map[string]*Port) // key = COMx
)

func getPort(com string, opts ...PortOption) (*Port, error) {
	// 检查端口是否存在
	portMutex.RLock()
	port, exists := portMap[com]
	portMutex.RUnlock()

	if exists {
		port.users.Add(1)
		return port, nil
	}

	portMutex.Lock()
	defer portMutex.Unlock()
	// 避免并发创建
	if port, exists = portMap[com]; exists {
		port.users.Add(1)
		return port, nil
	}

	port = &Port{}
	if err := port.Open(com, opts...); err != nil {
		return nil, err
	}
	portMap[com] = port
	return port, nil
}


func closePort(com string) error {
	portMutex.RLock()
	port, ok := portMap[com]
	portMutex.RUnlock()
	if !ok {
		return nil
	}

	if err := port.Close(); err != nil {
		return err
	}

	portMutex.Lock()
	defer portMutex.Unlock()
	if p, ok := portMap[com]; ok && p.users.Load() == 0 {
		delete(portMap, com)
	}
	return nil
}

type Device struct {
	port   *Port
	com    string
	closed atomic.Bool // 防止重复关闭
}

func NewDevice(com string, opts ...PortOption) (*Device, error) {
	port, err := getPort(com, opts...)
	if err != nil {
		return nil, err
	}
	return &Device{
		port: port,
		com:  com,
	}, nil
}

func (d *Device) Close() error {
	if d.closed.Load() {
		return nil // 防止重复关闭
	}
	d.closed.Store(true)
	return closePort(d.com) 
}

func (device *Device) ReadHoldingRegisters(id byte, address, quantity uint16) ([]byte, error) {
	return device.port.ReadHoldingRegisters(id, address, quantity)
}

func (device *Device) ReadInputRegisters(id byte, address, quantity uint16) ([]byte, error) {
	return device.port.ReadInputRegisters(id, address, quantity)
}

func (device *Device) WriteSingleRegister(id byte, address, value uint16) ([]byte, error) {
	return device.port.WriteSingleRegister(id, address, value)
}

func (device *Device) WriteMultipleRegisters(id byte, address, quantity uint16, value []byte) ([]byte, error) {
	return device.port.WriteMultipleRegisters(id, address, quantity, value)
}

func (device *Device) ReadCoils(id byte, address, quantity uint16) ([]byte, error) {
	return device.port.ReadCoils(id, address, quantity)
}

func (device *Device) ReadDiscreteInputs(id byte, address, quantity uint16) ([]byte, error) {
	return device.port.ReadDiscreteInputs(id, address, quantity)
}

func (device *Device) WriteSingleCoil(id byte, address, value uint16) ([]byte, error) {
	return device.port.WriteSingleCoil(id, address, value)
}

func (device *Device) WriteMultipleCoils(id byte, address, quantity uint16, value []byte) ([]byte, error) {
	return device.port.WriteMultipleCoils(id, address, quantity, value)
}

// 字节转换
func BytesToUint16(b []byte) []uint16 {
	if len(b)%2 != 0 {
		return nil
	}
	regs := make([]uint16, len(b)/2)
	for i := range regs {
		regs[i] = uint16(b[i*2])<<8 | uint16(b[i*2+1])
	}
	return regs
}

func Uint16ToUint32(regs []uint16) []uint32 {
	if len(regs)%2 != 0 {
		return nil
	}
	vals := make([]uint32, len(regs)/2)
	for i := range vals {
		vals[i] = uint32(regs[2*i])<<16 | uint32(regs[2*i+1])
	}
	return vals
}

func BytesToUint32(b []byte) []uint32 {
	if len(b)%4 != 0 {
		return nil
	}
	return Uint16ToUint32(BytesToUint16(b))
}
