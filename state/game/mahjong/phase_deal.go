package mahjong

import (
	"bytes"
	"fmt"

	"github.com/ratel-online/server/database"
	"github.com/ratel-online/server/rule"
)

// 发牌阶段
func (g *Mahjong) dealPhase(player *database.Player, game *database.Mahjong) error {
	// 初始化麻将规则
	mjRule := &rule.MahjongRule{}

	// 初始化牌墙
	tileWall := mjRule.InitTileWall()
	game.TileWall = tileWall

	// 确定庄家（第一个玩家为庄家）
	game.DealerIndex = 0
	game.CurrentIndex = 0

	// 发牌
	numPlayers := len(game.Players)
	hands, remainingTiles := mjRule.DealTiles(tileWall, numPlayers)

	// 设置玩家手牌
	for i := 0; i < numPlayers; i++ {
		game.Players[i].HandTiles = hands[i]
		if i == game.DealerIndex && len(hands[i]) > 0 {
			// 庄家第14张牌视为“摸到的牌”
			lastTile := hands[i][len(hands[i])-1]
			game.Players[i].LastDrawnTile = &lastTile
		}
		sortTiles(game.Players[i].HandTiles) // 理牌
		game.Players[i].IsDealer = (i == game.DealerIndex)
		game.Players[i].SeatWind = i // 简化的门风设置
	}

	// 更新剩余牌墙
	game.TileWall = remainingTiles

	// 发送初始信息给所有玩家
	for i, p := range game.Players {
		playerObj := database.GetPlayer(p.ID)
		if playerObj != nil {
			buf := bytes.Buffer{}
			buf.WriteString("=== 麻将游戏开始 ===\n")
			buf.WriteString(fmt.Sprintf("你是第 %d 位玩家\n", i+1))
			buf.WriteString(fmt.Sprintf("你的风位: %s\n", getWindName(p.SeatWind)))
			buf.WriteString(fmt.Sprintf("庄家: %s\n", game.Players[game.DealerIndex].Name))
			buf.WriteString(fmt.Sprintf("你的手牌: %s\n", tilesToString(p.HandTiles)))
			buf.WriteString("等待其他玩家行动...\n")
			_ = playerObj.WriteString(buf.String())
		}
	}

	// 开始游戏循环 - 庄家起手14张牌，直接打牌；其他人摸一打一
	// 庄家直接进入打牌阶段
	nextPlayer := game.Players[game.CurrentIndex]
	nextPlayerObj := database.GetPlayer(nextPlayer.ID)
	if nextPlayerObj != nil {
		nextPlayerObj.WriteString("轮到你打牌！\n")
		nextPlayer.State <- stateDiscard
	}

	return nil
}
