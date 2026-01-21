package mahjong

import (
	"fmt"

	"github.com/ratel-online/server/consts"
	"github.com/ratel-online/server/database"
)

// 胡牌阶段
func (g *Mahjong) winPhase(player *database.Player, game *database.Mahjong) error {
	// 显示胡牌结果
	if game.WinningInfo != nil {
		msg := fmt.Sprintf("\n=== 本局结束 ===\n")
		msg += fmt.Sprintf("胡牌玩家: %s\n", game.Players[game.WinningInfo.WinnerIndex].Name)
		msg += fmt.Sprintf("胡牌类型: %s\n", getWinTypeName(game.WinningInfo.WinType))
		msg += fmt.Sprintf("胡牌牌: %s\n", database.TileToString(game.WinningInfo.WinningTile))
		msg += fmt.Sprintf("总番数: %d\n", game.WinningInfo.TotalFan)
		msg += fmt.Sprintf("各玩家得分变化: %v\n", game.WinningInfo.ScoreChange)
		msg += fmt.Sprintf("各玩家本局得分: %v\n", game.RoundScores)
		msg += fmt.Sprintf("各玩家总局得分: %v\n", game.TotalScores)

		database.Broadcast(player.RoomID, msg)
	}

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
