package mahjong

import (
	"fmt"

	"github.com/ratel-online/server/consts"
	"github.com/ratel-online/server/database"
	"github.com/ratel-online/server/rule"
)

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
