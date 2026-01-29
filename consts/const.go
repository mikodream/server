package consts

import (
	"time"

	"github.com/ratel-online/core/consts"
)

type StateID int

const (
	_ StateID = iota
	StateWelcome
	StateHome
	StateJoin
	StateCreate
	StateWaiting
	StateGame
	StateRunFastGame
	StateUnoGame
	StateMahjongGame
	StateTexasGame
)

type SkillID int

const (
	_ SkillID = iota - 1
	SkillWYSS
	SkillHYJJ
	SkillGHJM
	SkillPFCZ
	SkillDHXJ
	SkillLJFZ
	SkillZWZB
	SkillSKLF
	Skill996
	SkillTZJW
)

const (
	IsStart = consts.IsStart
	IsStop  = consts.IsStop

	MinPlayers = 3
	// MaxPlayers https://github.com/ratel-online/server/issues/14 小鄧修改
	MaxPlayers = 3

	RoomStateWaiting = 1
	RoomStateRunning = 2

	GameTypeClassic = 1
	GameTypeLaiZi   = 2
	GameTypeSkill   = 3
	GameTypeRunFast = 4
	GameTypeTexas   = 5
	GameTypeMahjong = 6
	GameTypeUno     = 7

	RobTimeout         = 20 * time.Second
	PlayTimeout        = 40 * time.Second
	PlayMahjongTimeout = 30 * time.Second
	BetTimeout         = 60 * time.Second

	// Mahjong constants
	MIN_PLAYERS_MAHJONG = 2
	MAX_PLAYERS_MAHJONG = 4

	// 麻将牌类型
	TILE_WAN  = 0 // 万
	TILE_TIAO = 1 // 条
	TILE_TONG = 2 // 筒
	TILE_FENG = 3 // 风牌
	TILE_JIAN = 4 // 箭牌

	// 风牌值
	WIND_DONG = 1 // 东风
	WIND_NAN  = 2 // 南风
	WIND_XI   = 3 // 西风
	WIND_BEI  = 4 // 北风

	// 箭牌值
	JIAN_ZHONG = 5 // 红中
	JIAN_FA    = 6 // 发财
	JIAN_BAI   = 7 // 白板

	// 游戏状态
	GAME_STATUS_WAITING = 0
	GAME_STATUS_DEALING = 1
	GAME_STATUS_PLAYING = 2
	GAME_STATUS_END     = 3

	// 风圈
	WIND_ROUND_DONG = 0 // 东风圈
	WIND_ROUND_NAN  = 1 // 南风圈
	WIND_ROUND_XI   = 2 // 西风圈
	WIND_ROUND_BEI  = 3 // 北风圈

	// 操作类型
	ACTION_HU   = 0 // 胡
	ACTION_GANG = 1 // 杠
	ACTION_PENG = 2 // 碰
	ACTION_CHI  = 3 // 吃
	ACTION_PASS = 4 // 过

	// 胡牌类型
	WIN_TYPE_ZIMO         = 0 // 自摸
	WIN_TYPE_DIANGPAO     = 1 // 点炮
	WIN_TYPE_GANGSHANGHUA = 2 // 杠上花
	WIN_TYPE_GANGSHANGPAO = 3 // 杠上炮
	WIN_TYPE_QIANGGANGHU  = 4 // 抢杠胡

	// 组合类型
	SET_CHI      = 0 // 顺子
	SET_PENG     = 1 // 刻子
	SET_MINGGANG = 2 // 明杠
	SET_BUGANG   = 3 // 补杠

	// 番种
	FAN_BASIC_WIN    = 1 // 基础胡 1番
	FAN_MENQING      = 1 // 门清 1番
	FAN_DUANYAOJIU   = 1 // 断幺九 1番
	FAN_PINGHU       = 1 // 平胡 1番
	FAN_DUIDUIHU     = 2 // 对对胡 2番
	FAN_QIDUIZI      = 2 // 七对子 2番
	FAN_HUNYISE      = 2 // 混一色 2番
	FAN_QINGYISE     = 4 // 清一色 4番
	FAN_DASANYUAN    = 8 // 大三元 8番
	FAN_XIAOSIXI     = 6 // 小四喜 6番
	FAN_DASIXI       = 8 // 大四喜 8番
	FAN_SHISANYAO    = 8 // 十三幺 8番
	FAN_GANGSHANGHUA = 1 // 杠上花 1番
	FAN_ZIMO         = 1 // 自摸 1番
	FAN_FENGKE       = 1 // 风刻 1番
	FAN_JIANKE       = 1 // 箭刻 1番

	// 默认基础分数
	DEFAULT_BASE_SCORE = 1
)

// Room properties.
const (
	RoomPropsDotShuffle = "ds"
	RoomPropsLaiZi      = "lz"
	RoomPropsSkill      = "sk"
	RoomPropsPassword   = "pwd"
	RoomPropsPlayerNum  = "pn"
	RoomPropsChat       = "ct"
	RoomPropsShowIP     = "ip"
)

var MnemonicSorted = []int{15, 14, 2, 1, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3}

var RunFastMnemonicSorted = []int{2, 1, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3}

type Error struct {
	Code int
	Msg  string
	Exit bool
}

func (e Error) Error() string {
	return e.Msg
}

func NewErr(code int, exit bool, msg string) Error {
	return Error{Code: code, Exit: exit, Msg: msg}
}

var (
	ErrorsExist                   = NewErr(1, true, "Exist. ")
	ErrorsChanClosed              = NewErr(1, true, "Chan closed. ")
	ErrorsTimeout                 = NewErr(1, false, "Timeout. ")
	ErrorsInputInvalid            = NewErr(1, false, "Input invalid. ")
	ErrorsChatUnopened            = NewErr(1, false, "Chat disabled. ")
	ErrorsChatUnopenedDuringGame  = NewErr(1, false, "Chat disabled during game. ")
	ErrorsAuthFail                = NewErr(1, true, "Auth fail. ")
	ErrorsRoomInvalid             = NewErr(1, true, "Room invalid. ")
	ErrorsGameTypeInvalid         = NewErr(1, false, "Game type invalid. ")
	ErrorsRoomPlayersIsFull       = NewErr(1, false, "Room players is fill. ")
	ErrorsRoomPassword            = NewErr(1, false, "Sorry! Password incorrect! ")
	ErrorsJoinFailForRoomRunning  = NewErr(1, false, "Join fail, room is running. ")
	ErrorsJoinFailForKicked       = NewErr(1, false, "Join fail, you have been kicked from this room. ")
	ErrorsGamePlayersInvalid      = NewErr(1, false, "Game players invalid. ")
	ErrorsPokersFacesInvalid      = NewErr(1, false, "Pokers faces invalid. ")
	ErrorsHaveToPlay              = NewErr(1, false, "Have to play. ")
	ErrorsMustHaveToPlay          = NewErr(1, false, "There is a hand that can be played and must be played. ")
	ErrorsEndToPlay               = NewErr(1, false, "Can only come out at the end. ")
	ErrorsUnknownTexasRound       = NewErr(1, false, "Unknown texas round. ")
	ErrorsGamePlayersInsufficient = NewErr(1, false, "Game players insufficient. ")
	ErrorsCannotKickYourself      = NewErr(1, false, "Cannot kick yourself. ")
	ErrorsPlayerNotInRoom         = NewErr(1, true, "Player not in room. ")
	GameTypes                     = map[int]string{
		GameTypeClassic: "斗地主",
		GameTypeLaiZi:   "斗地主-癞子版",
		GameTypeSkill:   "斗地主-大招版",
		GameTypeRunFast: "跑得快",
		GameTypeTexas:   "德州扑克",
		//GameTypeUno:     "Uno",
		//GameTypeMahjong: "Mahjong",

	}
	GameTypesIds = []int{
		GameTypeClassic,
		GameTypeLaiZi,
		GameTypeSkill,
		GameTypeRunFast,
		GameTypeTexas,
		//GameTypeMahjong,
	}
	RoomStates = map[int]string{
		RoomStateWaiting: "Waiting",
		RoomStateRunning: "Running",
	}
)
