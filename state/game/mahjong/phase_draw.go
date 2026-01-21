package mahjong

import (
	"fmt"

	"github.com/ratel-online/server/consts"
	"github.com/ratel-online/server/database"
	"github.com/ratel-online/server/rule"
)

// 摸牌阶段
func (g *Mahjong) drawPhase(player *database.Player, game *database.Mahjong) error {
	// 检查是否还有牌可摸
	if len(game.TileWall) == 0 {
		// 牌墙已空，流局
		game.GameStatus = consts.GAME_STATUS_END
		for _, p := range game.Players {
			p.State <- stateFlow
		}
		return nil
	}

	// 从牌墙摸一张牌
	drawnTile := game.TileWall[0]
	game.TileWall = game.TileWall[1:]

	// 找到当前玩家
	currentPlayerIndex := game.GetPlayerIndex(player.ID)
	if currentPlayerIndex == -1 {
		return consts.ErrorsPlayerNotInRoom
	}

	// 加入手牌并排序
	game.Players[currentPlayerIndex].HandTiles = append(game.Players[currentPlayerIndex].HandTiles, drawnTile)
	sortTiles(game.Players[currentPlayerIndex].HandTiles)

	player.WriteString(fmt.Sprintf("你摸到了: %s\n", database.TileToString(drawnTile)))
	player.WriteString(fmt.Sprintf("当前手牌: %s\n", tilesToString(game.Players[currentPlayerIndex].HandTiles)))

	// 检查是否可以自摸胡牌
	mjRule := &rule.MahjongRule{}
	canWin, fans := mjRule.CanWin(
		removeLastElement(game.Players[currentPlayerIndex].HandTiles),
		drawnTile,
		true,
	)

	if canWin {
		// 设置胡牌信息
		game.WinningInfo = &database.WinningInfo{
			WinnerIndex: currentPlayerIndex,
			WinningTile: drawnTile,
			WinType:     consts.WIN_TYPE_ZIMO,
			FanInfo:     map[string]int{"自摸": 1},
			TotalFan:    sumFanValues(fans),
		}

		// 计算得分
		baseScore := consts.DEFAULT_BASE_SCORE
		winScore := mjRule.CalculateScore(fans, baseScore, consts.WIN_TYPE_ZIMO)

		// 更新分数
		for i := range game.Players {
			if i == currentPlayerIndex {
				// 胡牌玩家得分
				game.RoundScores[i] += winScore * (len(game.Players) - 1)
				game.TotalScores[i] += winScore * (len(game.Players) - 1)
			} else {
				// 其他玩家扣分
				game.RoundScores[i] -= winScore
				game.TotalScores[i] -= winScore
			}
		}

		// 通知所有玩家胡牌结果
		resultMsg := fmt.Sprintf("%s 自摸胡牌！\n", game.Players[currentPlayerIndex].Name)
		resultMsg += fmt.Sprintf("胡牌牌: %s\n", database.TileToString(drawnTile))
		resultMsg += fmt.Sprintf("番型: %v\n", fans)
		resultMsg += fmt.Sprintf("得分: %d\n", winScore)
		resultMsg += fmt.Sprintf("各玩家本局得分: %v\n", game.RoundScores)

		database.Broadcast(player.RoomID, resultMsg)

		// 转到胡牌阶段
		for _, p := range game.Players {
			p.State <- stateWin
		}
		return nil
	}

	// 检查是否可以暗杠
	concealedKongs := mjRule.CanAnGang(game.Players[currentPlayerIndex].HandTiles)
	if len(concealedKongs) > 0 {
		// 有暗杠可操作，进入操作阶段
		game.Players[currentPlayerIndex].Actions = []database.Action{}
		for _, kongTile := range concealedKongs {
			game.Players[currentPlayerIndex].Actions = append(game.Players[currentPlayerIndex].Actions, database.Action{
				ActionType: consts.ACTION_GANG,
				Tile:       kongTile,
				ExtraData:  "暗杠",
			})
		}

		// 同时检查是否可以补杠
		for _, exposedSet := range game.Players[currentPlayerIndex].ExposedSets {
			if exposedSet.SetType == consts.SET_PENG { // 已经碰的牌
				if mjRule.IsSameTile(exposedSet.Tiles[0], drawnTile) {
					game.Players[currentPlayerIndex].Actions = append(game.Players[currentPlayerIndex].Actions, database.Action{
						ActionType: consts.ACTION_GANG,
						Tile:       drawnTile,
						ExtraData:  "补杠",
					})
				}
			}
		}

		if len(game.Players[currentPlayerIndex].Actions) > 0 {
			player.WriteString("你可以进行杠牌操作:\n")
			for i, action := range game.Players[currentPlayerIndex].Actions {
				player.WriteString(fmt.Sprintf("%d. 杠 %s (%s)\n", i+1, database.TileToString(action.Tile), action.ExtraData.(string)))
			}
			player.WriteString("0. 不杠，直接出牌\n")

			game.Players[currentPlayerIndex].State <- stateAction
			return nil
		}
	}

	// 直接进入打牌阶段
	game.Players[currentPlayerIndex].State <- stateDiscard
	return nil
}
