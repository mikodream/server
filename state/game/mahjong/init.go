package mahjong

import (
	"github.com/ratel-online/server/consts"
	"github.com/ratel-online/server/database"
)

// InitMahjongGame 初始化麻将游戏
func InitMahjongGame(room *database.Room) (*database.Mahjong, error) {
	// 获取房间内的玩家
	roomPlayersMap := database.RoomPlayers(room.ID)

	// 检查玩家数量
	if len(roomPlayersMap) < consts.MIN_PLAYERS_MAHJONG || len(roomPlayersMap) > consts.MAX_PLAYERS_MAHJONG {
		return nil, consts.ErrorsGamePlayersInvalid
	}

	players := make([]*database.MahjongPlayer, 0)

	for playerId := range roomPlayersMap {
		player := database.GetPlayer(playerId)
		if player != nil {
			// 创建麻将玩家对象
			mahjongPlayer := &database.MahjongPlayer{
				ID:             player.ID,
				Name:           player.Name,
				HandTiles:      []database.MahjongTile{},
				ExposedSets:    []database.ExposedSet{},
				ConcealedKongs: []database.MahjongTile{},
				WinningTile:    database.MahjongTile{},
				IsDealer:       false,
				SeatWind:       0,
				Score:          int(player.Amount),
				Ready:          false,
				Actions:        []database.Action{},
				State:          make(chan int, 1),
			}
			players = append(players, mahjongPlayer)
		}
	}

	// 初始化游戏状态
	mahjongGame := &database.Mahjong{
		Room:         room,
		Players:      players,
		WindRound:    consts.WIND_ROUND_DONG, // 东风圈开始
		DealerIndex:  0,
		CurrentIndex: 0,
		TileWall:     []database.MahjongTile{},
		FlowerTiles:  []database.MahjongTile{},
		DiscardPile:  []database.DiscardRecord{},
		GameRound:    1,
		RoundScores:  make([]int, len(players)),
		TotalScores:  make([]int, len(players)),
		WinningInfo:  nil,
		GameStatus:   consts.GAME_STATUS_WAITING,
	}

	// 开始发牌阶段
	if len(players) > 0 {
		mahjongGame.Players[0].State <- 1 // stateDeal = 1
	}

	return mahjongGame, nil
}
