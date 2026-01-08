/*
 * @Author: zzh
 * @Date: 2025-09-15 08:56:57
 * @LastEditors: zzh
 * @LastEditTime: 2025-09-27 10:22:22
 * @Description: 终端相关 API
 */
package terminal

/*
#ifdef _WIN32
#include <windows.h>
#else
#include <stdio.h>
#include <termios.h>
#include <unistd.h>
#endif

void flush_input_buffer() {
#ifdef _WIN32
    // Windows系统：清空控制台输入缓冲区
    HANDLE hStdin = GetStdHandle(STD_INPUT_HANDLE);
    if (hStdin != INVALID_HANDLE_VALUE) {
        FlushConsoleInputBuffer(hStdin);
    }
#else
    // Unix-like系统：使用tcflush清空标准输入
    tcflush(STDIN_FILENO, TCIFLUSH);
#endif
}
*/
import "C"

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func CheckSN(sn string) error {
	if sn == "" {
		return fmt.Errorf("SN 不能为空")
	}
	//2 位年 +2 位月+8 位产品编码 + 6位序列号 = 18位
	if len(sn) != 18 {
		return fmt.Errorf("SN 长度错误")
	}
	// if !regexp.MustCompile(`^A5\d{2}(0[1-9]|[1-4]\d|5[0-3])C\d{5}$`).MatchString(sn) {
	// 	return fmt.Errorf("SN 格式错误")
	// }
	return nil
}

func WaitInputSN() string {
	for {
		sn := WaitInputString("请输入 SN 码：")
		err := CheckSN(sn)
		if err == nil {
			return sn
		} else {
			fmt.Printf("%s\n", err)
		}
	}
}

func WaitInputString(note string) string {
	fmt.Print(note)
	C.flush_input_buffer()
	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadString('\n')
	if err == nil {
		line = strings.Trim(line, "\r\n")
		return line
	}
	return ""
}

func WaitInputYorN(note string) string {
	for {
		cmd := WaitInputString(note)
		switch cmd {
		case "Y", "y", "N", "n":
			return cmd
		default:
			fmt.Println("请输入 Y 或者 N")
		}
	}
}

func WaitInputEnter(note string) {
	for {
		str := WaitInputString(note)
		if str == "" {
			return
		}
	}
}

func WaitInputPDU() byte {
	for {
		input := WaitInputString("请输入PDU类型（普通PDU为0，智能PDU为1）：")
		switch input {
		case "0":
			return 0
		case "1":
			return 1
		default:
			fmt.Println("请输入 0 或者 1")
		}
	}
}

func WaitExit(note string) {
	fmt.Println(note)
	fmt.Print("输入回车退出...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	os.Exit(1)
}
