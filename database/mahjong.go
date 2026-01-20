package database

// Mahjong 麻将游戏结构体
type Mahjong struct {
	Room         *Room            `json:"room"`
	Players      []*MahjongPlayer `json:"players"`
	WindRound    int              `json:"windRound"`    // 风圈: 0-东风圈, 1-南风圈, 2-西风圈, 3-北风圈
	DealerIndex  int              `json:"dealerIndex"`  // 庄家索引
	CurrentIndex int              `json:"currentIndex"` // 当前玩家索引
	TileWall     []MahjongTile    `json:"tileWall"`     // 牌墙（剩余牌）
	FlowerTiles  []MahjongTile    `json:"flowerTiles"`  // 花牌（本项目无花牌，保留接口）
	DiscardPile  []DiscardRecord  `json:"discardPile"`  // 弃牌堆
	GameRound    int              `json:"gameRound"`    // 当前局数
	RoundScores  []int            `json:"roundScores"`  // 各玩家本局得分
	TotalScores  []int            `json:"totalScores"`  // 各玩家总得分
	WinningInfo  *WinningInfo     `json:"winningInfo"`  // 胡牌信息
	GameStatus   int              `json:"gameStatus"`   // 游戏状态
}

// MahjongPlayer 麻将玩家结构体
type MahjongPlayer struct {
	ID             int64         `json:"id"`
	Name           string        `json:"name"`
	HandTiles      []MahjongTile `json:"handTiles"`      // 手牌
	ExposedSets    []ExposedSet  `json:"exposedSets"`    // 明刻/明杠/顺子等副露牌
	ConcealedKongs []MahjongTile `json:"concealedKongs"` // 暗杠牌
	WinningTile    MahjongTile   `json:"winningTile"`    // 和牌的牌
	IsDealer       bool          `json:"isDealer"`       // 是否为庄家
	SeatWind       int           `json:"seatWind"`       // 门风
	Score          int           `json:"score"`          // 当前分数
	Ready          bool          `json:"ready"`          // 是否听牌
	Actions        []Action      `json:"actions"`        // 可执行的操作
	State          chan int      `json:"state"`          // 玩家状态通道
}

// MahjongTile 麻将牌结构体
type MahjongTile struct {
	ID    int `json:"id"`    // 牌的唯一标识
	Value int `json:"value"` // 牌的数值 (1-9)
	Type  int `json:"type"`  // 牌的类型: 0-万, 1-条, 2-筒, 3-风牌, 4-箭牌
}

// ExposedSet 明刻/明杠/顺子等副露牌组合
type ExposedSet struct {
	SetType int           `json:"setType"` // 组合类型: 0-顺子, 1-刻子, 2-明杠, 3-补杠
	Tiles   []MahjongTile `json:"tiles"`   // 牌组合
	FromWho int           `json:"fromWho"` // 来源玩家索引
}

// DiscardRecord 弃牌记录
type DiscardRecord struct {
	PlayerIndex int         `json:"playerIndex"` // 弃牌玩家索引
	Tile        MahjongTile `json:"tile"`        // 弃牌
	Sequence    int         `json:"sequence"`    // 弃牌顺序
}

// WinningInfo 胡牌信息
type WinningInfo struct {
	WinnerIndex int            `json:"winnerIndex"` // 胡牌玩家索引
	WinningTile MahjongTile    `json:"winningTile"` // 和牌
	WinType     int            `json:"winType"`     // 胡牌类型: 0-自摸, 1-点炮, 2-杠上花, 3-杠上炮, 4-抢杠胡
	FanInfo     map[string]int `json:"fanInfo"`     // 番种信息
	TotalFan    int            `json:"totalFan"`    // 总番数
	ScoreChange []int          `json:"scoreChange"` // 各玩家分数变化
	IsGameOver  bool           `json:"isGameOver"`  // 是否结束本局
}

// Action 玩家可执行的操作
type Action struct {
	ActionType int         `json:"actionType"` // 操作类型: 0-胡, 1-杠, 2-碰, 3-吃, 4-跳过
	FromPlayer int         `json:"fromPlayer"` // 来源玩家索引
	Tile       MahjongTile `json:"tile"`       // 相关牌
	ExtraData  interface{} `json:"extraData"`  // 额外数据
}

// Clean 清理游戏资源
func (g *Mahjong) Clean() {
	if g != nil {
		// 关闭相关通道或其他资源清理
	}
}

// Player 获取指定ID的玩家
func (g *Mahjong) Player(id int64) *MahjongPlayer {
	for _, p := range g.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// NextPlayerIndex 获取下一个玩家索引
func (g *Mahjong) NextPlayerIndex(currentIndex int) int {
	return (currentIndex + 1) % len(g.Players)
}

// PrevPlayerIndex 获取上一个玩家索引
func (g *Mahjong) PrevPlayerIndex(currentIndex int) int {
	return (currentIndex - 1 + len(g.Players)) % len(g.Players)
}

// GetPlayerIndex 获取玩家在游戏中的索引
func (g *Mahjong) GetPlayerIndex(id int64) int {
	for i, p := range g.Players {
		if p.ID == id {
			return i
		}
	}
	return -1
}

// IsSameTile 判断两张牌是否相同
func IsSameTile(tile1, tile2 MahjongTile) bool {
	return tile1.Type == tile2.Type && tile1.Value == tile2.Value
}

// TileToString 将麻将牌转换为字符串表示
func TileToString(tile MahjongTile) string {
	tileTypeStr := []string{"万", "条", "筒", "风", "箭"}[tile.Type]
	if tile.Type >= 3 { // 风牌或箭牌
		switch tile.Value {
		case 1:
			return "东"
		case 2:
			return "南"
		case 3:
			return "西"
		case 4:
			return "北"
		case 5:
			return "中"
		case 6:
			return "发"
		case 7:
			return "白"
		default:
			return "风"
		}
	} else {
		return string(rune('0'+tile.Value)) + tileTypeStr
	}
}
