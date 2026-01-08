/*
 * @Author: zzh
 * @Date: 2025-10-29 10:11:02
 * @LastEditors: zzh
 * @LastEditTime: 2025-11-03 10:19:57
 * @Description: 请填写简介
 */
package util

import (
	"fmt"
	"sync/atomic"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"golang.org/x/sys/windows/registry"
)

// Ping 测试与目标IP的网络连通性，收到第一个成功响应后立即返回
func Ping(ip string) error {
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		return fmt.Errorf("new pinger failed: %w", err)
	}

	pinger.Count = 4
	pinger.Timeout = 5 * time.Second
	pinger.SetPrivileged(true) // window 需要特权模式发送ICMP包

	var received atomic.Bool
	pinger.OnRecv = func(pkt *probing.Packet) {
		// 仅在第一次收到响应时终止Ping过程
		if !received.Load() {
			received.Store(true)
			pinger.Stop()
		}
	}

	// 执行Ping检测（会阻塞直到完成或被Stop()终止）
	if err := pinger.Run(); err != nil {
		return fmt.Errorf("ping Failed: %w", err)
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv > 0 {
		fmt.Printf("Ping %s success, send %d, recv %d, time %.2f ms\n",
			ip, stats.PacketsSent, stats.PacketsRecv, stats.AvgRtt.Seconds()*1000)
		return nil
	}

	fmt.Printf("ping %s not recv reply\n", ip)
	return fmt.Errorf("ping not recv reply")
}

// 通过 windows 注册表获取本地端口
func ScanLocalPorts() ([]string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`HARDWARE\DEVICEMAP\SERIALCOMM`, registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	defer k.Close()

	names, err := k.ReadValueNames(-1)
	if err != nil {
		return nil, err
	}
	var list []string
	for _, vn := range names {
		port, _, err := k.GetStringValue(vn)
		if err != nil {
			continue
		}
		if port != "" {
			list = append(list, port)
		}
	}
	// fmt.Println(list)
	return list, nil
}
