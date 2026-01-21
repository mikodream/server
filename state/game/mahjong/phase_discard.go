package mahjong

import (
	"fmt"

	"github.com/ratel-online/server/consts"
	"github.com/ratel-online/server/database"
	"github.com/ratel-online/server/rule"
)

// 打牌阶段
func (g *Mahjong) discardPhase(player *database.Player, game *database.Mahjong) error {
	// 显示当前游戏状态信息
	player.WriteString("=== 当前游戏状态 ===\n")

	// 显示所有玩家的副露信息
	for i, p := range game.Players {
		player.WriteString(fmt.Sprintf("%s(%d号位)副露: ", p.Name, i+1))
		if len(p.ExposedSets) == 0 {
			player.WriteString("无\n")
		} else {
			for j, set := range p.ExposedSets {
				if j > 0 {
					player.WriteString(" ")
				}
				setStr := ""
				switch set.SetType {
				case consts.SET_CHI:
					setStr = "吃:" + tilesToString(set.Tiles)
				case consts.SET_PENG:
					setStr = "碰:" + tilesToString(set.Tiles)
				case consts.SET_MINGGANG:
					setStr = "明杠:" + tilesToString([]database.MahjongTile{set.Tiles[0]})
				case consts.SET_BUGANG:
					setStr = "补杠:" + tilesToString([]database.MahjongTile{set.Tiles[0]})
				default:
					setStr = tilesToString(set.Tiles)
				}
				player.WriteString(setStr)
			}
			player.WriteString("\n")
		}
	}

	// 显示牌河（弃牌堆）
	player.WriteString("牌河: ")
	if len(game.DiscardPile) == 0 {
		player.WriteString("无\n")
	} else {
		for _, discard := range game.DiscardPile {
			player.WriteString(database.TileToString(discard.Tile) + " ")
		}
		player.WriteString("\n")
	}

	currentPlayer := game.Players[game.GetPlayerIndex(player.ID)]
	if currentPlayer.LastDrawnTile != nil {

		player.WriteString(fmt.Sprintf("\n你的手牌: %s", tilesToString(currentPlayer.HandTiles)))
		player.WriteString(fmt.Sprintf("\n你摸到了: %s\n", database.TileToString(*currentPlayer.LastDrawnTile)))
		currentPlayer.LastDrawnTile = nil // 显示后清除
	}

	player.WriteString("\n请选择要打出的牌:\n")
	player.WriteString("你的手牌: ")

	for i, tile := range currentPlayer.HandTiles {
		player.WriteString(fmt.Sprintf("%d.%s ", i+1, database.TileToString(tile)))
	}
	player.WriteString("\n")

	player.StartTransaction()
	defer player.StopTransaction()

	tileIndex, err := askForIntWithRetry(player, consts.PlayMahjongTimeout)
	if err != nil {
		if err == consts.ErrorsTimeout {
			player.WriteString("操作超时，自动打出最后一张牌\n")
		}
		// 超时或退出，默认打最后一张牌
		tileIndex = len(game.Players[game.GetPlayerIndex(player.ID)].HandTiles)
	}

	tileIndex-- // 转换为数组索引

	if tileIndex < 0 || tileIndex >= len(game.Players[game.GetPlayerIndex(player.ID)].HandTiles) {
		// 无效选择，默认打最后一张牌
		tileIndex = len(game.Players[game.GetPlayerIndex(player.ID)].HandTiles) - 1
		player.WriteString("无效选择，打出最后一张牌\n")
	}

	// 获取要打的牌
	discardTile := game.Players[game.GetPlayerIndex(player.ID)].HandTiles[tileIndex]

	// 从手牌中移除该牌
	newHand := []database.MahjongTile{}
	for i, tile := range game.Players[game.GetPlayerIndex(player.ID)].HandTiles {
		if i != tileIndex {
			newHand = append(newHand, tile)
		}
	}
	game.Players[game.GetPlayerIndex(player.ID)].HandTiles = newHand

	// 添加到弃牌堆
	seq := len(game.DiscardPile)
	game.DiscardPile = append(game.DiscardPile, database.DiscardRecord{
		PlayerIndex: game.GetPlayerIndex(player.ID),
		Tile:        discardTile,
		Sequence:    seq,
	})

	// 通知所有玩家打牌信息
	discardMsg := fmt.Sprintf("%s 打出了 %s\n", player.Name, database.TileToString(discardTile))
	database.Broadcast(player.RoomID, discardMsg)

	// 检查是否流局
	mjRule := &rule.MahjongRule{}
	if mjRule.IsFlow(game.TileWall) {
		for _, p := range game.Players {
			p.State <- stateFlow
		}
		return nil
	}

	// 检查其他玩家是否可以对该牌进行操作（胡、杠、碰、吃）
	// 从下家开始检查，按照胡->杠->碰->吃的优先级
	currentPlayerIndex := game.GetPlayerIndex(player.ID)
	prevPlayerIndex := currentPlayerIndex // 当前玩家即为出牌玩家

	// 检查所有其他玩家是否可以操作这张牌
	for i := 0; i < len(game.Players)-1; i++ { // 减1是因为不包括自己
		nextCheckIndex := game.NextPlayerIndex(prevPlayerIndex)
		if nextCheckIndex == currentPlayerIndex {
			// 跳过出牌玩家自己
			break
		}

		checkingPlayer := game.Players[nextCheckIndex]
		checkingPlayerObj := database.GetPlayer(checkingPlayer.ID)

		if checkingPlayerObj != nil {
			// 检查该玩家是否可以对此牌进行操作
			mjRule := &rule.MahjongRule{}

			// 检查胡牌（点炮）
			canWin, fans := mjRule.CanWin(checkingPlayer.HandTiles, discardTile, false)
			if canWin {
				// 该玩家可以胡牌
				game.CurrentIndex = nextCheckIndex
				checkingPlayerObj.WriteString(fmt.Sprintf("检测到可胡牌: %s\n", database.TileToString(discardTile)))
				checkingPlayerObj.WriteString("1. 胡牌 0. 不胡\n")

				checkingPlayerObj.StartTransaction()
				defer checkingPlayerObj.StopTransaction()

				choice, err := askForIntWithRetry(checkingPlayerObj, consts.PlayMahjongTimeout)
				if err != nil || choice != 1 {
					if err == consts.ErrorsTimeout {
						checkingPlayerObj.WriteString("操作超时，跳过胡牌\n")
					} else {
						checkingPlayerObj.WriteString("跳过胡牌\n")
					}
				} else {
					// 执行胡牌操作
					winScore := mjRule.CalculateScore(fans, consts.DEFAULT_BASE_SCORE, consts.WIN_TYPE_DIANGPAO)

					// 更新分数
					game.RoundScores[nextCheckIndex] += winScore
					game.TotalScores[nextCheckIndex] += winScore
					game.RoundScores[currentPlayerIndex] -= winScore
					game.TotalScores[currentPlayerIndex] -= winScore

					// 设置胡牌信息
					game.WinningInfo = &database.WinningInfo{
						WinnerIndex: nextCheckIndex,
						WinningTile: discardTile,
						WinType:     consts.WIN_TYPE_DIANGPAO,
						FanInfo:     map[string]int{"点炮胡": 1},
						TotalFan:    sumFanValues(fans),
					}

					resultMsg := fmt.Sprintf("%s 接炮胡牌！\n", game.Players[nextCheckIndex].Name)
					resultMsg += fmt.Sprintf("点炮者: %s\n", game.Players[currentPlayerIndex].Name)
					resultMsg += fmt.Sprintf("胡牌牌: %s\n", database.TileToString(discardTile))
					resultMsg += fmt.Sprintf("得分: %d\n", winScore)
					resultMsg += fmt.Sprintf("各玩家本局得分: %v\n", game.RoundScores)

					database.Broadcast(player.RoomID, resultMsg)

					// 转到胡牌阶段
					for _, p := range game.Players {
						p.State <- stateWin
					}
					return nil
				}
			}

			// 检查杠牌
			canGang, _ := mjRule.CanGang(checkingPlayer.HandTiles, checkingPlayer.ExposedSets, discardTile)
			if canGang {
				// 该玩家可以杠牌
				game.CurrentIndex = nextCheckIndex
				checkingPlayerObj.WriteString(fmt.Sprintf("检测到可杠牌: %s\n", database.TileToString(discardTile)))
				checkingPlayerObj.WriteString("1. 杠牌 0. 不杠\n")

				checkingPlayerObj.StartTransaction()
				defer checkingPlayerObj.StopTransaction()

				choice, err := askForIntWithRetry(checkingPlayerObj, consts.PlayMahjongTimeout)
				if err != nil || choice != 1 {
					if err == consts.ErrorsTimeout {
						checkingPlayerObj.WriteString("操作超时，跳过杠牌\n")
					} else {
						checkingPlayerObj.WriteString("跳过杠牌\n")
					}
				} else {
					// 执行杠牌操作
					// 从手牌中移除3张相同的牌
					handTiles := []database.MahjongTile{}
					count := 0
					for _, tile := range checkingPlayer.HandTiles {
						if !mjRule.IsSameTile(tile, discardTile) || count >= 3 {
							handTiles = append(handTiles, tile)
						} else {
							count++
						}
					}

					game.Players[nextCheckIndex].HandTiles = handTiles

					// 添加到明杠组合
					gangSet := database.ExposedSet{
						SetType: consts.SET_MINGGANG,
						Tiles:   []database.MahjongTile{discardTile, discardTile, discardTile, discardTile},
						FromWho: currentPlayerIndex,
					}
					game.Players[nextCheckIndex].ExposedSets = append(game.Players[nextCheckIndex].ExposedSets, gangSet)

					// 从牌墙补一张牌
					if len(game.TileWall) > 0 {
						bonusTile := game.TileWall[0]
						game.TileWall = game.TileWall[1:]
						game.Players[nextCheckIndex].HandTiles = append(game.Players[nextCheckIndex].HandTiles, bonusTile)
						sortTiles(game.Players[nextCheckIndex].HandTiles)
						game.Players[nextCheckIndex].LastDrawnTile = &bonusTile

						// 检查补杠后是否胡牌（杠上花）
						canWin, fans := mjRule.CanWin(
							removeLastElement(game.Players[nextCheckIndex].HandTiles),
							bonusTile,
							true,
						)

						if canWin {
							winScore := mjRule.CalculateScore(fans, consts.DEFAULT_BASE_SCORE, consts.WIN_TYPE_GANGSHANGHUA)

							// 更新分数
							for i := range game.Players {
								if i == nextCheckIndex {
									game.RoundScores[i] += winScore * (len(game.Players) - 1)
									game.TotalScores[i] += winScore * (len(game.Players) - 1)
								} else {
									game.RoundScores[i] -= winScore
									game.TotalScores[i] -= winScore
								}
							}

							game.WinningInfo = &database.WinningInfo{
								WinnerIndex: nextCheckIndex,
								WinningTile: bonusTile,
								WinType:     consts.WIN_TYPE_GANGSHANGHUA,
								FanInfo:     map[string]int{"杠上花": 1},
								TotalFan:    sumFanValues(fans),
							}

							resultMsg := fmt.Sprintf("%s 杠上开花！\n", game.Players[nextCheckIndex].Name)
							resultMsg += fmt.Sprintf("杠上花得分: %d\n", winScore)
							resultMsg += fmt.Sprintf("各玩家本局得分: %v\n", game.RoundScores)

							database.Broadcast(player.RoomID, resultMsg)

							for _, p := range game.Players {
								p.State <- stateWin
							}
							return nil
						}
					}

					// 杠牌后轮次变为杠牌者，继续打牌
					game.CurrentIndex = nextCheckIndex
					checkingPlayer.State <- stateDiscard
					return nil
				}
			}

			// 检查碰牌
			if mjRule.CanPeng(checkingPlayer.HandTiles, discardTile) {
				// 该玩家可以碰牌
				game.CurrentIndex = nextCheckIndex
				checkingPlayerObj.WriteString(fmt.Sprintf("检测到可碰牌: %s\n", database.TileToString(discardTile)))
				checkingPlayerObj.WriteString("1. 碰牌 0. 不碰\n")

				checkingPlayerObj.StartTransaction()
				defer checkingPlayerObj.StopTransaction()

				choice, err := askForIntWithRetry(checkingPlayerObj, consts.PlayMahjongTimeout)
				if err != nil || choice != 1 {
					if err == consts.ErrorsTimeout {
						checkingPlayerObj.WriteString("操作超时，跳过碰牌\n")
					} else {
						checkingPlayerObj.WriteString("跳过碰牌\n")
					}
				} else {
					// 执行碰牌操作
					// 从手牌中移除2张相同的牌
					handTiles := []database.MahjongTile{}
					count := 0
					for _, tile := range checkingPlayer.HandTiles {
						if !mjRule.IsSameTile(tile, discardTile) || count >= 2 {
							handTiles = append(handTiles, tile)
						} else {
							count++
						}
					}

					game.Players[nextCheckIndex].HandTiles = handTiles

					// 添加到碰牌组合
					pengSet := database.ExposedSet{
						SetType: consts.SET_PENG,
						Tiles:   []database.MahjongTile{discardTile, discardTile, discardTile},
						FromWho: currentPlayerIndex,
					}
					game.Players[nextCheckIndex].ExposedSets = append(game.Players[nextCheckIndex].ExposedSets, pengSet)

					// 碰牌后需要立即打牌
					game.CurrentIndex = nextCheckIndex
					checkingPlayer.State <- stateDiscard
					return nil
				}
			}

			// 检查吃牌（只有下家可以吃）
			if nextCheckIndex == game.NextPlayerIndex(currentPlayerIndex) {
				possibleChis := mjRule.CanChi(checkingPlayer.HandTiles, discardTile)
				if len(possibleChis) > 0 {
					// 该玩家可以吃牌
					game.CurrentIndex = nextCheckIndex
					checkingPlayerObj.WriteString(fmt.Sprintf("检测到可吃牌: %s\n", database.TileToString(discardTile)))
					for i, chi := range possibleChis {
						checkingPlayerObj.WriteString(fmt.Sprintf("%d. 吃 %s\n", i+1, tilesToString(chi)))
					}
					checkingPlayerObj.WriteString("0. 不吃\n")

					checkingPlayerObj.StartTransaction()
					defer checkingPlayerObj.StopTransaction()

					choice, err := askForIntWithRetry(checkingPlayerObj, consts.PlayMahjongTimeout)
					if err != nil || choice == 0 || choice > len(possibleChis) {
						if err == consts.ErrorsTimeout {
							checkingPlayerObj.WriteString("操作超时，跳过吃牌\n")
						} else {
							checkingPlayerObj.WriteString("跳过吃牌\n")
						}
					} else {
						// 执行吃牌操作
						selectedChi := possibleChis[choice-1]
						action := database.Action{
							ActionType: consts.ACTION_CHI,
							Tile:       discardTile,
							ExtraData:  selectedChi,
						}
						game.CurrentIndex = nextCheckIndex
						return g.executeAction(checkingPlayerObj, game, action, currentPlayerIndex)
					}
				}
			}

			prevPlayerIndex = nextCheckIndex
		}
	}

	// 没有任何玩家操作这张牌，轮到下家摸牌
	game.CurrentIndex = game.NextPlayerIndex(game.CurrentIndex)
	nextPlayer := game.Players[game.CurrentIndex]
	nextPlayerObj := database.GetPlayer(nextPlayer.ID)
	if nextPlayerObj != nil {
		nextPlayer.State <- stateDraw
	}

	return nil
}
