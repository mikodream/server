package mahjong

import (
	"time"

	"github.com/ratel-online/core/log"
	"github.com/ratel-online/server/consts"
	"github.com/ratel-online/server/database"
)

type Mahjong struct{}

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
