package mahjong

import (
	"github.com/ratel-online/server/consts"
	"github.com/ratel-online/server/database"
)

// 辅助函数：获取胡牌类型名称
func getWinTypeName(winType int) string {
	switch winType {
	case consts.WIN_TYPE_ZIMO:
		return "自摸"
	case consts.WIN_TYPE_DIANGPAO:
		return "点炮"
	case consts.WIN_TYPE_GANGSHANGHUA:
		return "杠上花"
	case consts.WIN_TYPE_GANGSHANGPAO:
		return "杠上炮"
	case consts.WIN_TYPE_QIANGGANGHU:
		return "抢杠胡"
	default:
		return "未知"
	}
}

// 辅助函数：按优先级排序操作
func sortActionsByPriority(actions []database.Action) []database.Action {
	// 优先级：胡 > 杠 > 碰 > 吃
	priority := func(actionType int) int {
		switch actionType {
		case consts.ACTION_HU:
			return 4
		case consts.ACTION_GANG:
			return 3
		case consts.ACTION_PENG:
			return 2
		case consts.ACTION_CHI:
			return 1
		default:
			return 0
		}
	}

	for i := 0; i < len(actions); i++ {
		for j := i + 1; j < len(actions); j++ {
			if priority(actions[i].ActionType) < priority(actions[j].ActionType) {
				actions[i], actions[j] = actions[j], actions[i]
			}
		}
	}

	return actions
}
