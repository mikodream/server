package mahjong

import (
	"time"

	"github.com/ratel-online/server/consts"
	"github.com/ratel-online/server/database"
)

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

// askForIntWithRetry 安全地获取整数输入，排除非超时的错误输入
func askForIntWithRetry(player *database.Player, timeout ...time.Duration) (int, error) {
	for {
		val, err := player.AskForInt(timeout...)
		if err == nil {
			return val, nil
		}
		// 如果是超时或其他需要退出的错误，则直接返回
		if err == consts.ErrorsTimeout || err == consts.ErrorsExist || err == consts.ErrorsChanClosed {
			return 0, err
		}
		// 非数字输入，提示并重新尝试
		player.WriteString("输入无效，请输入数字：")
	}
}
