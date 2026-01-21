package mahjong

import (
	"fmt"

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

	//// 询问是否继续游戏
	//database.Broadcast(player.RoomID, "是否继续游戏？(y/n)\n")
	//
	//// 这里应该收集玩家的选择，为了简化直接继续
	//// 实际实现中需要等待所有玩家确认
	//
	//// 重新开始游戏
	//game.GameStatus = consts.GAME_STATUS_DEALING
	//game.WinningInfo = nil
	//
	//// 庄家轮转
	//game.DealerIndex = game.NextPlayerIndex(game.DealerIndex)
	//game.CurrentIndex = game.DealerIndex
	//
	//// 重新初始化玩家状态
	//for _, p := range game.Players {
	//	p.HandTiles = []database.MahjongTile{}
	//	p.ExposedSets = []database.ExposedSet{}
	//	p.Actions = []database.Action{}
	//}
	//
	//// 开始新一局
	//firstPlayer := game.Players[game.DealerIndex]
	//firstPlayerObj := database.GetPlayer(firstPlayer.ID)
	//if firstPlayerObj != nil {
	//	firstPlayer.State <- stateDeal
	//}

	return nil
}
