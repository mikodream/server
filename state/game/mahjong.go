package game

import (
	"github.com/ratel-online/server/database"
	"github.com/ratel-online/server/state/game/mahjong"
)

// 初始化麻将游戏
func InitMahjongGame(room *database.Room) (*database.Mahjong, error) {
	return mahjong.InitMahjongGame(room)
}
