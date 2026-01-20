package mahjong

import (
	"bytes"
	"fmt"
	"time"

	"github.com/ratel-online/core/log"
	"github.com/ratel-online/server/consts"
	"github.com/ratel-online/server/database"
	"github.com/ratel-online/server/rule"
)

type Mahjong struct{}

var (
	stateDeal    = 1 // 发牌阶段
	stateDraw    = 2 // 摸牌阶段
	stateAction  = 3 // 操作阶段
	stateDiscard = 4 // 打牌阶段
	stateWin     = 5 // 胡牌阶段
	stateFlow    = 6 // 流局阶段
	stateWaiting = 7 // 等待阶段
)

func (g *Mahjong) Next(player *database.Player) (consts.StateID, error) {
	room := database.GetRoom(player.RoomID)
	if room == nil {
		return 0, player.WriteError(consts.ErrorsExist)
	}
	game := room.Game.(*database.Mahjong)

	loopCount := 0
	for {
		loopCount++
		if loopCount%100 == 0 {
			log.Infof("[Mahjong.Next] Player %d (Room %d) loop count: %d, room.State: %d\n", player.ID, player.RoomID, loopCount, room.State)
		}
		if room.State == consts.RoomStateWaiting {
			log.Infof("[Mahjong.Next] Player %d exiting, room state changed to waiting, loop count: %d\n", player.ID, loopCount)
			return consts.StateWaiting, nil
		}
		mahjongPlayer := game.Player(player.ID)
		if mahjongPlayer == nil {
			return 0, player.WriteError(consts.ErrorsExist)
		}

		select {
		case state, ok := <-mahjongPlayer.State:
			if !ok {
				return 0, consts.ErrorsChanClosed
			}
			switch state {
			case stateDeal:
				err := g.dealPhase(player, game)
				if err != nil {
					log.Error(err)
					return 0, err
				}
			case stateDraw:
				err := g.drawPhase(player, game)
				if err != nil {
					log.Error(err)
					return 0, err
				}
			case stateAction:
				err := g.actionPhase(player, game)
				if err != nil {
					log.Error(err)
					return 0, err
				}
			case stateDiscard:
				err := g.discardPhase(player, game)
				if err != nil {
					log.Error(err)
					return 0, err
				}
			case stateWin:
				err := g.winPhase(player, game)
				if err != nil {
					log.Error(err)
					return 0, err
				}
			case stateFlow:
				err := g.flowPhase(player, game)
				if err != nil {
					log.Error(err)
					return 0, err
				}
			case stateWaiting:
				return consts.StateWaiting, nil
			default:
				return 0, consts.ErrorsChanClosed
			}
		case <-time.After(5 * time.Second):
			// 防止通道阻塞导致的死锁
			return 0, consts.ErrorsTimeout
		}
	}
}

func (*Mahjong) Exit(player *database.Player) consts.StateID {
	return consts.StateHome
}

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

// 操作阶段（吃碰杠胡）
func (g *Mahjong) actionPhase(player *database.Player, game *database.Mahjong) error {
	// 检查是否有其他玩家对上家打出的牌可以操作
	prevPlayerIndex := game.PrevPlayerIndex(game.CurrentIndex)
	if prevPlayerIndex != game.CurrentIndex {
		// 获取上家最近打出的牌
		var lastDiscard database.MahjongTile
		found := false
		for i := len(game.DiscardPile) - 1; i >= 0; i-- {
			if game.DiscardPile[i].PlayerIndex == prevPlayerIndex {
				lastDiscard = game.DiscardPile[i].Tile
				found = true
				break
			}
		}

		if found {
			mjRule := &rule.MahjongRule{}
			currentPlayerIndex := game.GetPlayerIndex(player.ID)

			// 检查当前玩家是否可以对该牌操作
			possibleActions := []database.Action{}

			// 检查胡牌（点炮）
			canWin, fans := mjRule.CanWin(game.Players[currentPlayerIndex].HandTiles, lastDiscard, false)
			if canWin {
				possibleActions = append(possibleActions, database.Action{
					ActionType: consts.ACTION_HU,
					FromPlayer: prevPlayerIndex,
					Tile:       lastDiscard,
					ExtraData:  fans,
				})
			}

			// 检查杠牌
			canGang, gangType := mjRule.CanGang(game.Players[currentPlayerIndex].HandTiles, game.Players[currentPlayerIndex].ExposedSets, lastDiscard)
			if canGang {
				possibleActions = append(possibleActions, database.Action{
					ActionType: consts.ACTION_GANG,
					FromPlayer: prevPlayerIndex,
					Tile:       lastDiscard,
					ExtraData:  gangType,
				})
			}

			// 检查碰牌
			if mjRule.CanPeng(game.Players[currentPlayerIndex].HandTiles, lastDiscard) {
				possibleActions = append(possibleActions, database.Action{
					ActionType: consts.ACTION_PENG,
					FromPlayer: prevPlayerIndex,
					Tile:       lastDiscard,
				})
			}

			// 检查吃牌（只有下家可以吃）
			nextPlayerIndex := game.NextPlayerIndex(prevPlayerIndex)
			if currentPlayerIndex == nextPlayerIndex {
				possibleChis := mjRule.CanChi(game.Players[currentPlayerIndex].HandTiles, lastDiscard)
				for _, chi := range possibleChis {
					possibleActions = append(possibleActions, database.Action{
						ActionType: consts.ACTION_CHI,
						FromPlayer: prevPlayerIndex,
						Tile:       lastDiscard,
						ExtraData:  chi,
					})
				}
			}

			if len(possibleActions) > 0 {
				// 按照优先级排序：胡 > 杠 > 碰 > 吃
				sortedActions := sortActionsByPriority(possibleActions)

				// 通知玩家可执行的操作
				player.WriteString("检测到可执行操作:\n")
				for i, action := range sortedActions {
					actionName := getActionName(action.ActionType)
					player.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, actionName, database.TileToString(action.Tile)))
				}
				player.WriteString("0. 跳过\n")

				// 让玩家选择操作
				player.StartTransaction()
				defer player.StopTransaction()

				choice, err := player.AskForInt(consts.PlayMahjongTimeout)
				if err != nil {
					// 超时或错误，跳过操作
					player.WriteString("操作超时，自动跳过\n")
					game.CurrentIndex = game.NextPlayerIndex(game.CurrentIndex)
					nextPlayer := game.Players[game.CurrentIndex]
					nextPlayerObj := database.GetPlayer(nextPlayer.ID)
					if nextPlayerObj != nil {
						nextPlayer.State <- stateDraw
					}
					return nil
				}

				if choice == 0 || choice > len(sortedActions) {
					// 跳过操作
					player.WriteString("跳过操作\n")
					game.CurrentIndex = game.NextPlayerIndex(game.CurrentIndex)
					nextPlayer := game.Players[game.CurrentIndex]
					nextPlayerObj := database.GetPlayer(nextPlayer.ID)
					if nextPlayerObj != nil {
						nextPlayer.State <- stateDraw
					}
				} else {
					// 执行选定的操作
					selectedAction := sortedActions[choice-1]
					err := g.executeAction(player, game, selectedAction, prevPlayerIndex)
					if err != nil {
						return err
					}
				}
			} else {
				// 没有可执行的操作，继续游戏
				game.CurrentIndex = game.NextPlayerIndex(game.CurrentIndex)
				nextPlayer := game.Players[game.CurrentIndex]
				nextPlayerObj := database.GetPlayer(nextPlayer.ID)
				if nextPlayerObj != nil {
					nextPlayer.State <- stateDraw
				}
			}
		}
	}

	return nil
}

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

				player.WriteString(fmt.Sprintf("杠后补牌: %s\n", database.TileToString(bonusTile)))

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
			for _, tile := range game.Players[currentPlayerIndex].HandTiles {
				if !mjRule.IsSameTile(tile, action.Tile) {
					handTiles = append(handTiles, tile)
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

				player.WriteString(fmt.Sprintf("补杠后补牌: %s\n", database.TileToString(bonusTile)))

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
		handTiles := []database.MahjongTile{}
		usedCount := 0
		for _, tile := range game.Players[currentPlayerIndex].HandTiles {
			isUsed := false
			for _, chiTile := range chiTiles {
				if mjRule.IsSameTile(tile, chiTile) && !database.IsSameTile(tile, action.Tile) { // 不包括被吃的牌
					if usedCount < len(chiTiles)-1 { // 减1是因为被吃的牌不算在手牌中扣除
						isUsed = true
						usedCount++
						break
					}
				}
			}
			if !isUsed {
				handTiles = append(handTiles, tile)
			}
		}

		game.Players[currentPlayerIndex].HandTiles = handTiles

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

	player.WriteString("\n请选择要打出的牌:\n")
	player.WriteString("你的手牌: ")

	for i, tile := range game.Players[game.GetPlayerIndex(player.ID)].HandTiles {
		player.WriteString(fmt.Sprintf("%d.%s ", i+1, database.TileToString(tile)))
	}
	player.WriteString("\n")

	player.StartTransaction()
	defer player.StopTransaction()

	tileIndex, err := player.AskForInt(consts.PlayMahjongTimeout)
	if err != nil {
		// 超时，默认打最后一张牌
		tileIndex = len(game.Players[game.GetPlayerIndex(player.ID)].HandTiles)
		player.WriteString("操作超时，自动打出最后一张牌\n")
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

				choice, err := checkingPlayerObj.AskForInt(consts.PlayMahjongTimeout)
				if err != nil || choice != 1 {
					checkingPlayerObj.WriteString("跳过胡牌\n")
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

				choice, err := checkingPlayerObj.AskForInt(consts.PlayMahjongTimeout)
				if err != nil || choice != 1 {
					checkingPlayerObj.WriteString("跳过杠牌\n")
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

						checkingPlayerObj.WriteString(fmt.Sprintf("杠后补牌: %s\n", database.TileToString(bonusTile)))

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

				choice, err := checkingPlayerObj.AskForInt(consts.PlayMahjongTimeout)
				if err != nil || choice != 1 {
					checkingPlayerObj.WriteString("跳过碰牌\n")
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

					choice, err := checkingPlayerObj.AskForInt(consts.PlayMahjongTimeout)
					if err != nil || choice == 0 || choice > len(possibleChis) {
						checkingPlayerObj.WriteString("跳过吃牌\n")
					} else {
						// 执行吃牌操作
						selectedChi := possibleChis[choice-1]
						// 从手牌中移除用于吃的两张牌
						handTiles := []database.MahjongTile{}
						for _, tile := range checkingPlayer.HandTiles {
							isUsed := false
							for _, chiTile := range selectedChi {
								if mjRule.IsSameTile(tile, chiTile) && !database.IsSameTile(tile, discardTile) { // 不包括被吃的牌
									isUsed = true
									break
								}
							}
							if !isUsed {
								handTiles = append(handTiles, tile)
							}
						}

						game.Players[nextCheckIndex].HandTiles = handTiles

						// 添加到吃牌组合
						chiSet := database.ExposedSet{
							SetType: consts.SET_CHI,
							Tiles:   selectedChi,
							FromWho: currentPlayerIndex,
						}
						game.Players[nextCheckIndex].ExposedSets = append(game.Players[nextCheckIndex].ExposedSets, chiSet)

						// 吃牌后需要立即打牌
						game.CurrentIndex = nextCheckIndex
						checkingPlayer.State <- stateDiscard
						return nil
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

	// 询问是否继续游戏
	database.Broadcast(player.RoomID, "是否继续游戏？(y/n)\n")

	// 这里应该收集玩家的选择，为了简化直接继续
	// 实际实现中需要等待所有玩家确认

	// 重新开始游戏
	game.GameStatus = consts.GAME_STATUS_DEALING
	game.WinningInfo = nil

	// 庄家轮转
	game.DealerIndex = game.NextPlayerIndex(game.DealerIndex)
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

// 辅助函数：将牌数组转换为字符串
func tilesToString(tiles []database.MahjongTile) string {
	if len(tiles) == 0 {
		return "无"
	}

	result := ""
	for i, tile := range tiles {
		result += database.TileToString(tile)
		if i < len(tiles)-1 {
			result += " "
		}
	}
	return result
}

// 辅助函数：对手牌进行排序
func sortTiles(tiles []database.MahjongTile) {
	for i := 0; i < len(tiles); i++ {
		for j := i + 1; j < len(tiles); j++ {
			if shouldSwap(tiles[i], tiles[j]) {
				tiles[i], tiles[j] = tiles[j], tiles[i]
			}
		}
	}
}

// 辅助函数：判断是否需要交换位置
func shouldSwap(tile1, tile2 database.MahjongTile) bool {
	if tile1.Type != tile2.Type {
		return tile1.Type > tile2.Type
	}
	return tile1.Value > tile2.Value
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

// 辅助函数：移除切片最后一个元素并返回除最后一个元素外的所有元素
func removeLastElement(slice []database.MahjongTile) []database.MahjongTile {
	if len(slice) <= 1 {
		return []database.MahjongTile{}
	}
	return slice[:len(slice)-1]
}

// 辅助函数：计算番值总和
func sumFanValues(fans []int) int {
	sum := 0
	for _, fan := range fans {
		sum += fan
	}
	return sum
}
