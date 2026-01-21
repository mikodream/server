package mahjong

import (
	"fmt"

	"github.com/ratel-online/server/consts"
	"github.com/ratel-online/server/database"
	"github.com/ratel-online/server/rule"
)

// 执行具体操作
func (g *Mahjong) executeAction(player *database.Player, game *database.Mahjong, action database.Action, fromPlayerIndex int) error {
	currentPlayerIndex := game.GetPlayerIndex(player.ID)
	mjRule := &rule.MahjongRule{}

	switch action.ActionType {
	case consts.ACTION_HU:
		// 胡牌操作
		fans := action.ExtraData.([]int)
		winScore := mjRule.CalculateScore(fans, consts.DEFAULT_BASE_SCORE, consts.WIN_TYPE_DIANGPAO)

		// 更新分数
		game.RoundScores[currentPlayerIndex] += winScore
		game.TotalScores[currentPlayerIndex] += winScore
		game.RoundScores[fromPlayerIndex] -= winScore
		game.TotalScores[fromPlayerIndex] -= winScore

		// 设置胡牌信息
		game.WinningInfo = &database.WinningInfo{
			WinnerIndex: currentPlayerIndex,
			WinningTile: action.Tile,
			WinType:     consts.WIN_TYPE_DIANGPAO,
			FanInfo:     map[string]int{"点炮胡": 1},
			TotalFan:    sumFanValues(fans),
		}

		// 通知所有玩家胡牌结果
		resultMsg := fmt.Sprintf("%s 接炮胡牌！\n", game.Players[currentPlayerIndex].Name)
		resultMsg += fmt.Sprintf("点炮者: %s\n", game.Players[fromPlayerIndex].Name)
		resultMsg += fmt.Sprintf("胡牌牌: %s\n", database.TileToString(action.Tile))
		resultMsg += fmt.Sprintf("得分: %d\n", winScore)
		resultMsg += fmt.Sprintf("各玩家本局得分: %v\n", game.RoundScores)

		database.Broadcast(player.RoomID, resultMsg)

		// 转到胡牌阶段
		for _, p := range game.Players {
			p.State <- stateWin
		}
	case consts.ACTION_GANG:
		// 杠牌操作
		gangType := action.ExtraData.(int)

		switch gangType {
		case 2: // 明杠
			player.WriteString(fmt.Sprintf("你进行了明杠: %s\n", database.TileToString(action.Tile)))

			// 从手牌中移除3张相同的牌
			handTiles := []database.MahjongTile{}
			count := 0
			for _, tile := range game.Players[currentPlayerIndex].HandTiles {
				if !mjRule.IsSameTile(tile, action.Tile) || count >= 3 {
					handTiles = append(handTiles, tile)
				} else {
					count++
				}
			}

			game.Players[currentPlayerIndex].HandTiles = handTiles

			// 添加到明杠组合
			gangSet := database.ExposedSet{
				SetType: consts.SET_MINGGANG,
				Tiles:   []database.MahjongTile{action.Tile, action.Tile, action.Tile, action.Tile},
				FromWho: fromPlayerIndex,
			}
			game.Players[currentPlayerIndex].ExposedSets = append(game.Players[currentPlayerIndex].ExposedSets, gangSet)

			// 从牌墙补一张牌
			if len(game.TileWall) > 0 {
				bonusTile := game.TileWall[0]
				game.TileWall = game.TileWall[1:]
				game.Players[currentPlayerIndex].HandTiles = append(game.Players[currentPlayerIndex].HandTiles, bonusTile)
				sortTiles(game.Players[currentPlayerIndex].HandTiles)
				game.Players[currentPlayerIndex].LastDrawnTile = &bonusTile

				// 检查补杠后是否胡牌（杠上花）
				canWin, fans := mjRule.CanWin(
					removeLastElement(game.Players[currentPlayerIndex].HandTiles),
					bonusTile,
					true,
				)

				if canWin {
					winScore := mjRule.CalculateScore(fans, consts.DEFAULT_BASE_SCORE, consts.WIN_TYPE_GANGSHANGHUA)

					// 更新分数
					for i := range game.Players {
						if i == currentPlayerIndex {
							game.RoundScores[i] += winScore * (len(game.Players) - 1)
							game.TotalScores[i] += winScore * (len(game.Players) - 1)
						} else {
							game.RoundScores[i] -= winScore
							game.TotalScores[i] -= winScore
						}
					}

					game.WinningInfo = &database.WinningInfo{
						WinnerIndex: currentPlayerIndex,
						WinningTile: bonusTile,
						WinType:     consts.WIN_TYPE_GANGSHANGHUA,
						FanInfo:     map[string]int{"杠上花": 1},
						TotalFan:    sumFanValues(fans),
					}

					resultMsg := fmt.Sprintf("%s 杠上开花！\n", game.Players[currentPlayerIndex].Name)
					resultMsg += fmt.Sprintf("杠上花得分: %d\n", winScore)
					resultMsg += fmt.Sprintf("各玩家本局得分: %v\n", game.RoundScores)

					database.Broadcast(player.RoomID, resultMsg)

					for _, p := range game.Players {
						p.State <- stateWin
					}
					return nil
				}
			}

			// 杠后轮次不变，继续当前玩家
			game.Players[currentPlayerIndex].State <- stateDiscard
		case 3: // 补杠
			player.WriteString(fmt.Sprintf("你进行了补杠: %s\n", database.TileToString(action.Tile)))

			// 从手牌中移除1张牌
			handTiles := []database.MahjongTile{}
			count := 0
			for _, tile := range game.Players[currentPlayerIndex].HandTiles {
				if !mjRule.IsSameTile(tile, action.Tile) || count >= 1 {
					handTiles = append(handTiles, tile)
				} else {
					count++
				}
			}
			game.Players[currentPlayerIndex].HandTiles = handTiles

			// 找到对应的碰牌组合，升级为补杠
			for i, set := range game.Players[currentPlayerIndex].ExposedSets {
				if set.SetType == consts.SET_PENG && mjRule.IsSameTile(set.Tiles[0], action.Tile) {
					game.Players[currentPlayerIndex].ExposedSets[i] = database.ExposedSet{
						SetType: consts.SET_BUGANG,
						Tiles:   []database.MahjongTile{action.Tile, action.Tile, action.Tile, action.Tile},
						FromWho: set.FromWho,
					}
					break
				}
			}

			// 从牌墙补一张牌
			if len(game.TileWall) > 0 {
				bonusTile := game.TileWall[0]
				game.TileWall = game.TileWall[1:]
				game.Players[currentPlayerIndex].HandTiles = append(game.Players[currentPlayerIndex].HandTiles, bonusTile)
				sortTiles(game.Players[currentPlayerIndex].HandTiles)
				game.Players[currentPlayerIndex].LastDrawnTile = &bonusTile

				// 检查补杠后是否胡牌（杠上花）
				canWin, fans := mjRule.CanWin(
					removeLastElement(game.Players[currentPlayerIndex].HandTiles),
					bonusTile,
					true,
				)

				if canWin {
					winScore := mjRule.CalculateScore(fans, consts.DEFAULT_BASE_SCORE, consts.WIN_TYPE_GANGSHANGHUA)

					// 更新分数
					for i := range game.Players {
						if i == currentPlayerIndex {
							game.RoundScores[i] += winScore * (len(game.Players) - 1)
							game.TotalScores[i] += winScore * (len(game.Players) - 1)
						} else {
							game.RoundScores[i] -= winScore
							game.TotalScores[i] -= winScore
						}
					}

					game.WinningInfo = &database.WinningInfo{
						WinnerIndex: currentPlayerIndex,
						WinningTile: bonusTile,
						WinType:     consts.WIN_TYPE_GANGSHANGHUA,
						FanInfo:     map[string]int{"杠上花": 1},
						TotalFan:    sumFanValues(fans),
					}

					resultMsg := fmt.Sprintf("%s 杠上开花！\n", game.Players[currentPlayerIndex].Name)
					resultMsg += fmt.Sprintf("杠上花得分: %d\n", winScore)
					resultMsg += fmt.Sprintf("各玩家本局得分: %v\n", game.RoundScores)

					database.Broadcast(player.RoomID, resultMsg)

					for _, p := range game.Players {
						p.State <- stateWin
					}
					return nil
				}
			}

			// 补杠后轮次不变，继续当前玩家
			game.Players[currentPlayerIndex].State <- stateDiscard
		}
	case consts.ACTION_PENG:
		// 碰牌操作
		player.WriteString(fmt.Sprintf("你进行了碰牌: %s\n", database.TileToString(action.Tile)))

		// 从手牌中移除2张相同的牌
		handTiles := []database.MahjongTile{}
		count := 0
		for _, tile := range game.Players[currentPlayerIndex].HandTiles {
			if !mjRule.IsSameTile(tile, action.Tile) || count >= 2 {
				handTiles = append(handTiles, tile)
			} else {
				count++
			}
		}

		game.Players[currentPlayerIndex].HandTiles = handTiles
		sortTiles(game.Players[currentPlayerIndex].HandTiles)

		// 添加到碰牌组合
		pengSet := database.ExposedSet{
			SetType: consts.SET_PENG,
			Tiles:   []database.MahjongTile{action.Tile, action.Tile, action.Tile},
			FromWho: fromPlayerIndex,
		}
		game.Players[currentPlayerIndex].ExposedSets = append(game.Players[currentPlayerIndex].ExposedSets, pengSet)

		// 碰牌后需要立即打牌
		game.Players[currentPlayerIndex].State <- stateDiscard
	case consts.ACTION_CHI:
		// 吃牌操作
		chiTiles := action.ExtraData.([]database.MahjongTile)
		player.WriteString(fmt.Sprintf("你进行了吃牌: %s\n", tilesToString(chiTiles)))

		// 从手牌中移除用于吃的两张牌
		// 找出手牌中需要被扣除的两张牌
		neededTiles := []database.MahjongTile{}
		for _, t := range chiTiles {
			if !mjRule.IsSameTile(t, action.Tile) {
				neededTiles = append(neededTiles, t)
			}
		}

		handTiles := []database.MahjongTile{}
		for _, tile := range game.Players[currentPlayerIndex].HandTiles {
			found := false
			for i, needed := range neededTiles {
				if mjRule.IsSameTile(tile, needed) {
					neededTiles = append(neededTiles[:i], neededTiles[i+1:]...)
					found = true
					break
				}
			}
			if !found {
				handTiles = append(handTiles, tile)
			}
		}

		game.Players[currentPlayerIndex].HandTiles = handTiles
		sortTiles(game.Players[currentPlayerIndex].HandTiles)

		// 添加到吃牌组合
		chiSet := database.ExposedSet{
			SetType: consts.SET_CHI,
			Tiles:   chiTiles,
			FromWho: fromPlayerIndex,
		}
		game.Players[currentPlayerIndex].ExposedSets = append(game.Players[currentPlayerIndex].ExposedSets, chiSet)

		// 吃牌后需要立即打牌
		game.Players[currentPlayerIndex].State <- stateDiscard
	}

	return nil
}
