package mahjong

import "github.com/ratel-online/server/database"

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

// 辅助函数：移除切片最后一个元素并返回除最后一个元素外的所有元素
func removeLastElement(slice []database.MahjongTile) []database.MahjongTile {
	if len(slice) <= 1 {
		return []database.MahjongTile{}
	}
	return slice[:len(slice)-1]
}
