/*
 * @Author: zzh
 * @Date: 2025-09-19 14:00:47
 * @LastEditors: zzh
 * @LastEditTime: 2025-12-08
 * @Description: 表格窗口 - 校准工具简化版
 */
package window

import (
	"main/gui/table"

	"fyne.io/fyne/v2"
)

type Tutorial struct {
	Title string
	View  func() fyne.CanvasObject
}

var (
	Tutorials = map[Navigate]Tutorial{
		NavCalibration: {
			Title: "校准流程",
			View:  table.GetCalibrationTable,
		},
	}
)
