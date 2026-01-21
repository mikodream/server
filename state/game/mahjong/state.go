package mahjong

const (
	stateDeal    = 1 // 发牌阶段
	stateDraw    = 2 // 摸牌阶段
	stateAction  = 3 // 操作阶段
	stateDiscard = 4 // 打牌阶段
	stateWin     = 5 // 胡牌阶段
	stateFlow    = 6 // 流局阶段
	stateWaiting = 7 // 等待阶段
)
