package mahjong

import (
	"fmt"

	"github.com/ratel-online/server/consts"
	"github.com/ratel-online/server/database"
)

// 流局阶段
func (g *Mahjong) flowPhase(player *database.Player, game *database.Mahjong) error {
	// 流局处理
	msg := "牌局流局！牌墙已空且无人胡牌。\n"
	msg += fmt.Sprintf("各玩家总得分: %v\n", game.TotalScores)

	database.Broadcast(player.RoomID, msg)

	// 重置本局得分
	for i := range game.RoundScores {
		game.RoundScores[i] = 0
	}

	// 标记房间为等待状态，这样 Mahjong.Next 循环会退出并回到 Waiting 状态
	game.Room.State = consts.RoomStateWaiting

	// 通知所有玩家退出当前游戏循环
	for _, p := range game.Players {
		select {
		case p.State <- stateWaiting:
		default:
			// 如果通道满了，忽略，反正 Room.State 已经改了，下次循环也会退出
		}
	}

	return nil
}
