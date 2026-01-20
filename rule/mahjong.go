package rule

import (
	"github.com/ratel-online/server/consts"
	"github.com/ratel-online/server/database"
	"math/rand"
	"time"
)

// MahjongRule 麻将规则
type MahjongRule struct{}

// 初始化麻将牌墙
func (m *MahjongRule) InitTileWall() []database.MahjongTile {
	var tiles []database.MahjongTile

	// 添加序数牌：万、条、筒，每种1-9各4张
	for tileType := consts.TILE_WAN; tileType <= consts.TILE_TONG; tileType++ {
		for value := 1; value <= 9; value++ {
			for i := 0; i < 4; i++ {
				tile := database.MahjongTile{
					ID:    len(tiles),
					Value: value,
					Type:  tileType,
				}
				tiles = append(tiles, tile)
			}
		}
	}

	// 添加字牌：东南西北中发白，各4张
	// 风牌 (东南西北)
	for value := consts.WIND_DONG; value <= consts.WIND_BEI; value++ {
		for i := 0; i < 4; i++ {
			tile := database.MahjongTile{
				ID:    len(tiles),
				Value: value,
				Type:  consts.TILE_FENG,
			}
			tiles = append(tiles, tile)
		}
	}

	// 箭牌 (中发白)
	for value := consts.JIAN_ZHONG; value <= consts.JIAN_BAI; value++ {
		for i := 0; i < 4; i++ {
			tile := database.MahjongTile{
				ID:    len(tiles),
				Value: value,
				Type:  consts.TILE_JIAN,
			}
			tiles = append(tiles, tile)
		}
	}

	// 洗牌
	rand.Seed(time.Now().UnixNano())
	for i := len(tiles) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		tiles[i], tiles[j] = tiles[j], tiles[i]
	}

	return tiles
}

// 发牌
func (m *MahjongRule) DealTiles(tileWall []database.MahjongTile, numPlayers int) ([][]database.MahjongTile, []database.MahjongTile) {
	hands := make([][]database.MahjongTile, numPlayers)

	// 庄家14张牌，其他人13张牌
	tileIndex := 0
	for i := 0; i < numPlayers; i++ {
		handSize := 13
		if i == 0 { // 庄家多一张牌
			handSize = 14
		}

		for j := 0; j < handSize; j++ {
			if tileIndex < len(tileWall) {
				hands[i] = append(hands[i], tileWall[tileIndex])
				tileIndex++
			}
		}
	}

	// 返回剩余的牌墙
	remainingTiles := make([]database.MahjongTile, len(tileWall)-tileIndex)
	copy(remainingTiles, tileWall[tileIndex:])

	return hands, remainingTiles
}

// 检查是否可以胡牌
func (m *MahjongRule) CanWin(hand []database.MahjongTile, winTile database.MahjongTile, isSelfDrawn bool) (bool, []int) {
	// 将胡牌加入手牌
	allTiles := append(append([]database.MahjongTile{}, hand...), winTile)

	// 检查基本胡牌牌型
	if m.checkBasicWin(allTiles) {
		// 计算番数
		fans := m.calculateFans(hand, winTile, isSelfDrawn, false)
		return len(fans) > 0, fans
	}

	return false, nil
}

// 检查基本胡牌牌型（4个面子+1个对子 或 七对子 或 十三幺）
func (m *MahjongRule) checkBasicWin(tiles []database.MahjongTile) bool {
	// 检查牌的数量是否正确
	if len(tiles)%3 != 2 {
		return false
	}

	// 统计各种牌的数量
	tileCounts := m.countTiles(tiles)

	// 检查七对子（只有14张牌的情况）
	if len(tiles) == 14 && m.checkSevenPairs(tileCounts) {
		return true
	}

	// 检查十三幺
	if m.checkThirteenOrphans(tiles) {
		return true
	}

	// 检查普通胡牌牌型（4个面子+1个对子）
	return m.checkNormalWin(tileCounts)
}

// 统计牌的数量
func (m *MahjongRule) countTiles(tiles []database.MahjongTile) map[string]int {
	counts := make(map[string]int)
	for _, tile := range tiles {
		key := m.getTileKey(tile)
		counts[key]++
	}
	return counts
}

// 获取牌的键值
func (m *MahjongRule) getTileKey(tile database.MahjongTile) string {
	return string(rune('0'+tile.Type)) + string(rune('0'+tile.Value))
}

// 检查七对子
func (m *MahjongRule) checkSevenPairs(tileCounts map[string]int) bool {
	pairCount := 0
	for _, count := range tileCounts {
		if count%2 != 0 {
			return false
		}
		if count == 2 {
			pairCount++
		}
	}
	return pairCount == 7
}

// 检查十三幺
func (m *MahjongRule) checkThirteenOrphans(tiles []database.MahjongTile) bool {
	requiredTiles := make(map[string]bool)

	// 添加必需的幺九牌
	for _, tileType := range []int{consts.TILE_WAN, consts.TILE_TIAO, consts.TILE_TONG} {
		requiredTiles[m.getTileKey(database.MahjongTile{Type: tileType, Value: 1})] = true
		requiredTiles[m.getTileKey(database.MahjongTile{Type: tileType, Value: 9})] = true
	}

	// 添加字牌
	for value := consts.WIND_DONG; value <= consts.WIND_BEI; value++ {
		requiredTiles[m.getTileKey(database.MahjongTile{Type: consts.TILE_FENG, Value: value})] = true
	}
	for value := consts.JIAN_ZHONG; value <= consts.JIAN_BAI; value++ {
		requiredTiles[m.getTileKey(database.MahjongTile{Type: consts.TILE_JIAN, Value: value})] = true
	}

	tileCounts := m.countTiles(tiles)
	hasPair := false

	for tileKey, count := range tileCounts {
		if !requiredTiles[tileKey] {
			return false
		}
		if count == 2 {
			if hasPair {
				// 已经有一个对子了，不能再有
				return false
			}
			hasPair = true
		} else if count != 1 {
			// 十三幺中每种牌最多只能有2张（一对），最少1张
			return false
		}
	}

	return hasPair && len(tileCounts) == 13
}

// 检查普通胡牌牌型（4个面子+1个对子）
func (m *MahjongRule) checkNormalWin(tileCounts map[string]int) bool {
	// 尝试移除一个对子，然后检查是否能组成4个面子
	for tileKey, count := range tileCounts {
		if count >= 2 {
			// 临时移除对子
			tempCounts := make(map[string]int)
			for k, v := range tileCounts {
				tempCounts[k] = v
			}
			tempCounts[tileKey] -= 2

			if m.canFormMelds(tempCounts) {
				return true
			}
		}
	}
	return false
}

// 检查是否能组成面子（顺子或刻子）
func (m *MahjongRule) canFormMelds(tileCounts map[string]int) bool {
	// 先处理刻子
	keys := make([]string, 0, len(tileCounts))
	for k := range tileCounts {
		keys = append(keys, k)
	}

	for _, key := range keys {
		count := tileCounts[key]
		if count >= 3 {
			// 形成刻子
			tileCounts[key] -= 3
			if m.canFormMelds(tileCounts) {
				return true
			}
			tileCounts[key] += 3 // 回退
		}
	}

	// 再处理顺子（只对数字牌有效）
	for _, key := range keys {
		tile := m.parseTileKey(key)
		if tile.Type <= consts.TILE_TONG && tileCounts[key] > 0 { // 只对万、条、筒处理顺子
			// 检查是否能形成顺子：tile, tile+1, tile+2
			next1Key := m.getTileKey(database.MahjongTile{Type: tile.Type, Value: tile.Value + 1})
			next2Key := m.getTileKey(database.MahjongTile{Type: tile.Type, Value: tile.Value + 2})

			if tileCounts[next1Key] > 0 && tileCounts[next2Key] > 0 {
				// 形成顺子
				tileCounts[key]--
				tileCounts[next1Key]--
				tileCounts[next2Key]--

				if m.canFormMelds(tileCounts) {
					return true
				}

				// 回退
				tileCounts[key]++
				tileCounts[next1Key]++
				tileCounts[next2Key]++
			}
		}
	}

	// 如果还有剩余的牌，则无法组成面子
	for _, count := range tileCounts {
		if count > 0 {
			return false
		}
	}

	return true
}

// 解析牌键值
func (m *MahjongRule) parseTileKey(key string) database.MahjongTile {
	if len(key) < 2 {
		return database.MahjongTile{}
	}

	tileType := int(key[0] - '0')
	tileValue := int(key[1] - '0')

	return database.MahjongTile{Type: tileType, Value: tileValue}
}

// 计算番数
func (m *MahjongRule) calculateFans(hand []database.MahjongTile, winTile database.MahjongTile, isSelfDrawn bool, isGangShangHua bool) []int {
	var fans []int

	// 基础胡牌 1番
	fans = append(fans, consts.FAN_BASIC_WIN)

	// 检查是否门清（没有吃碰杠）
	if m.isMenqing(hand) {
		fans = append(fans, consts.FAN_MENQING)

		// 门清自摸 1番
		if isSelfDrawn {
			fans = append(fans, consts.FAN_ZIMO)
		}
	}

	// 杠上花 1番
	if isGangShangHua {
		fans = append(fans, consts.FAN_GANGSHANGHUA)
	}

	// 检查断幺九
	if m.isDuanyaojiu(append(append([]database.MahjongTile{}, hand...), winTile)) {
		fans = append(fans, consts.FAN_DUANYAOJIU)
	}

	// 检查对对胡
	if m.isDuiduihu(append(append([]database.MahjongTile{}, hand...), winTile)) {
		fans = append(fans, consts.FAN_DUIDUIHU)
	}

	// 检查七对子
	if len(hand)+1 == 14 && m.checkSevenPairs(m.countTiles(append(append([]database.MahjongTile{}, hand...), winTile))) {
		fans = append(fans, consts.FAN_QIDUIZI)
	}

	// 检查清一色
	if m.isQingyise(append(append([]database.MahjongTile{}, hand...), winTile)) {
		fans = append(fans, consts.FAN_QINGYISE)
	} else if m.isHunyise(append(append([]database.MahjongTile{}, hand...), winTile)) {
		fans = append(fans, consts.FAN_HUNYISE)
	}

	// 检查风刻和箭刻
	tileCounts := m.countTiles(append(append([]database.MahjongTile{}, hand...), winTile))
	for tileKey, count := range tileCounts {
		if count >= 3 {
			tile := m.parseTileKey(tileKey)
			// 风刻
			if tile.Type == consts.TILE_FENG && tile.Value >= consts.WIND_DONG && tile.Value <= consts.WIND_BEI {
				fans = append(fans, consts.FAN_FENGKE)
			}
			// 箭刻
			if tile.Type == consts.TILE_JIAN && tile.Value >= consts.JIAN_ZHONG && tile.Value <= consts.JIAN_BAI {
				fans = append(fans, consts.FAN_JIANKE)
			}
		}
	}

	return fans
}

// 检查是否门清（没有吃碰杠）
func (m *MahjongRule) isMenqing(hand []database.MahjongTile) bool {
	// 实际游戏中需要根据玩家的副露牌来判断
	// 这里简化处理，如果手牌数量是14张（自摸）或13张（接炮）且能胡牌，则认为是门清
	// 更准确的实现需要访问玩家的ExposedSets字段
	return true
}

// 检查断幺九
func (m *MahjongRule) isDuanyaojiu(tiles []database.MahjongTile) bool {
	for _, tile := range tiles {
		// 数字牌中的1、9是幺九牌
		if (tile.Type <= consts.TILE_TONG) && (tile.Value == 1 || tile.Value == 9) {
			return false
		}
		// 字牌都是幺九牌
		if tile.Type >= consts.TILE_FENG {
			return false
		}
	}
	return true
}

// 检查对对胡
func (m *MahjongRule) isDuiduihu(tiles []database.MahjongTile) bool {
	tileCounts := m.countTiles(tiles)

	// 对对胡：由4个刻子/杠子 + 1个对子组成
	meldCount := 0
	pairFound := false

	for _, count := range tileCounts {
		if count == 3 || count == 4 {
			// 刻子或杠子
			meldCount++
		} else if count == 2 {
			// 对子
			if pairFound {
				// 已经有对子了，不能再有
				return false
			}
			pairFound = true
		} else if count == 1 {
			// 单张牌，不符合对对胡
			return false
		}
	}

	return meldCount == 4 && pairFound
}

// 检查清一色
func (m *MahjongRule) isQingyise(tiles []database.MahjongTile) bool {
	if len(tiles) == 0 {
		return false
	}

	firstType := tiles[0].Type
	for _, tile := range tiles {
		if tile.Type != firstType || tile.Type >= consts.TILE_FENG {
			return false
		}
	}
	return true
}

// 检查混一色
func (m *MahjongRule) isHunyise(tiles []database.MahjongTile) bool {
	if len(tiles) == 0 {
		return false
	}

	var suitType *int
	hasHonors := false

	for _, tile := range tiles {
		if tile.Type >= consts.TILE_FENG {
			// 字牌
			hasHonors = true
		} else {
			// 序数牌
			if suitType == nil {
				suitType = &tile.Type
			} else if *suitType != tile.Type {
				// 出现了两种不同的序数牌花色
				return false
			}
		}
	}

	// 必须有一种序数牌花色，并且可以有字牌
	return suitType != nil && hasHonors
}

// 检查是否可以吃牌
func (m *MahjongRule) CanChi(hand []database.MahjongTile, discardTile database.MahjongTile) [][]database.MahjongTile {
	var possibleChis [][]database.MahjongTile

	// 只有数字牌才能吃
	if discardTile.Type > consts.TILE_TONG {
		return possibleChis
	}

	// 统计手牌
	tileCounts := m.countTiles(hand)

	// 尝试组成顺子
	// 情况1: [n-2, n-1, n]
	if discardTile.Value >= 3 {
		key1 := m.getTileKey(database.MahjongTile{Type: discardTile.Type, Value: discardTile.Value - 2})
		key2 := m.getTileKey(database.MahjongTile{Type: discardTile.Type, Value: discardTile.Value - 1})

		if tileCounts[key1] > 0 && tileCounts[key2] > 0 {
			chi := []database.MahjongTile{
				discardTile,
				{Type: discardTile.Type, Value: discardTile.Value - 2},
				{Type: discardTile.Type, Value: discardTile.Value - 1},
			}
			possibleChis = append(possibleChis, chi)
		}
	}

	// 情况2: [n-1, n, n+1]
	if discardTile.Value >= 2 && discardTile.Value <= 8 {
		key1 := m.getTileKey(database.MahjongTile{Type: discardTile.Type, Value: discardTile.Value - 1})
		key2 := m.getTileKey(database.MahjongTile{Type: discardTile.Type, Value: discardTile.Value + 1})

		if tileCounts[key1] > 0 && tileCounts[key2] > 0 {
			chi := []database.MahjongTile{
				discardTile,
				{Type: discardTile.Type, Value: discardTile.Value - 1},
				{Type: discardTile.Type, Value: discardTile.Value + 1},
			}
			possibleChis = append(possibleChis, chi)
		}
	}

	// 情况3: [n, n+1, n+2]
	if discardTile.Value <= 7 {
		key1 := m.getTileKey(database.MahjongTile{Type: discardTile.Type, Value: discardTile.Value + 1})
		key2 := m.getTileKey(database.MahjongTile{Type: discardTile.Type, Value: discardTile.Value + 2})

		if tileCounts[key1] > 0 && tileCounts[key2] > 0 {
			chi := []database.MahjongTile{
				discardTile,
				{Type: discardTile.Type, Value: discardTile.Value + 1},
				{Type: discardTile.Type, Value: discardTile.Value + 2},
			}
			possibleChis = append(possibleChis, chi)
		}
	}

	return possibleChis
}

// 检查是否可以碰牌
func (m *MahjongRule) CanPeng(hand []database.MahjongTile, discardTile database.MahjongTile) bool {
	tileCounts := m.countTiles(hand)
	tileKey := m.getTileKey(discardTile)
	return tileCounts[tileKey] >= 2
}

// 检查是否可以杠牌
func (m *MahjongRule) CanGang(hand []database.MahjongTile, exposedSets []database.ExposedSet, discardTile database.MahjongTile) (bool, int) {
	tileCounts := m.countTiles(hand)
	tileKey := m.getTileKey(discardTile)

	// 明杠：手牌中有3张相同的牌，别人打出第4张
	if tileCounts[tileKey] == 3 {
		return true, 2 // 明杠
	}

	// 检查补杠：已经碰了牌，现在摸到或别人打出第4张
	for _, set := range exposedSets {
		if set.SetType == consts.SET_PENG { // 是碰的牌
			if len(set.Tiles) == 3 && m.IsSameTile(set.Tiles[0], discardTile) {
				return true, 3 // 补杠
			}
		}
	}

	return false, 0
}

// 检查是否可以暗杠
func (m *MahjongRule) CanAnGang(hand []database.MahjongTile) []database.MahjongTile {
	var possibleKongs []database.MahjongTile
	tileCounts := m.countTiles(hand)

	for tileKey, count := range tileCounts {
		if count == 4 {
			tile := m.parseTileKey(tileKey)
			possibleKongs = append(possibleKongs, tile)
		}
	}

	return possibleKongs
}

// 检查两张牌是否相同
func (m *MahjongRule) IsSameTile(tile1, tile2 database.MahjongTile) bool {
	return tile1.Type == tile2.Type && tile1.Value == tile2.Value
}

// 计算得分
func (m *MahjongRule) CalculateScore(fanList []int, baseScore int, winType int) int {
	totalFan := 0
	for _, fan := range fanList {
		totalFan += fan
	}

	score := baseScore * totalFan

	// 根据胡牌类型调整分数
	switch winType {
	case consts.WIN_TYPE_GANGSHANGHUA:
		score *= 2 // 杠上花翻倍
	case consts.WIN_TYPE_GANGSHANGPAO:
		score *= 2 // 杠上炮翻倍
	case consts.WIN_TYPE_QIANGGANGHU:
		score *= 2 // 抢杠胡翻倍
	}

	return score
}

// 检查是否流局
func (m *MahjongRule) IsFlow(tileWall []database.MahjongTile) bool {
	// 当牌墙剩余牌数≤4张时，未有人胡牌则判定流局
	return len(tileWall) <= 4
}
