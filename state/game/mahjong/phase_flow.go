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

	// 询问是否继续游戏
	database.Broadcast(player.RoomID, "是否继续游戏？(y/n)\n")

	// 为了简化，直接重新开始游戏
	game.GameStatus = consts.GAME_STATUS_DEALING
	game.WinningInfo = nil

	// 庄家保持不变
	game.CurrentIndex = game.DealerIndex

	// 重新初始化玩家状态
	for _, p := range game.Players {
		p.HandTiles = []database.MahjongTile{}
		p.ExposedSets = []database.ExposedSet{}
		p.Actions = []database.Action{}
	}

	// 开始新一局
	firstPlayer := game.Players[game.DealerIndex]
	firstPlayerObj := database.GetPlayer(firstPlayer.ID)
	if firstPlayerObj != nil {
		firstPlayer.State <- stateDeal
	}

	return nil
}
