package mahjong

import "github.com/ratel-online/server/consts"

// 辅助函数：获取动作名称
func getActionName(actionType int) string {
	switch actionType {
	case consts.ACTION_HU:
		return "胡"
	case consts.ACTION_GANG:
		return "杠"
	case consts.ACTION_PENG:
		return "碰"
	case consts.ACTION_CHI:
		return "吃"
	default:
		return "未知"
	}
}

// 辅助函数：获取风位名称
func getWindName(wind int) string {
	switch wind {
	case 0:
		return "东风"
	case 1:
		return "南风"
	case 2:
		return "西风"
	case 3:
		return "北风"
	default:
		return "未知风"
	}
}

// 辅助函数：计算番值总和
func sumFanValues(fans []int) int {
	sum := 0
	for _, fan := range fans {
		sum += fan
	}
	return sum
}
