package sz

import (
	"common/logs"
	"common/utils"
	"encoding/json"
	"framework/remote"
	"game/compone/base"
	"game/compone/proto"
	"github.com/jinzhu/copier"
	"time"
)

// GameFrame 具体的游戏类型
type GameFrame struct {
	room       base.RoomFrame
	gameRule   proto.GameRule
	gameData   *GameData
	logic      *Logic
	gameResult *GameResult // 游戏结果
}

func (g GameFrame) GetGameData(session *remote.Session) any {
	// 通过session和room拿到当前用户
	user := g.room.GetUsers()[session.GetUid()]
	// 判断当前用户是否已经看牌，如果已经看牌则返回牌数据，但是看不到其它用户的手牌
	// 深拷贝
	var gameData GameData
	copier.CopyWithOption(&gameData, g.gameData, copier.Option{DeepCopy: true})
	for i := 0; i < g.gameData.ChairCount; i++ {
		if g.gameData.HandCards[i] != nil {
			gameData.HandCards[i] = make([]int, 3)
		} else {
			gameData.HandCards[i] = nil // 没有手牌的用户进入房间是不会显示自己的牌
		}
	}
	if g.gameData.LookCards[user.ChairID] == 1 {
		// 当前用户已经看牌, 只显示他自己的牌，其他用户的牌他还是看不见
		gameData.HandCards[user.ChairID] = g.gameData.HandCards[user.ChairID]
	}
	return gameData
}

// ServerMessagePush 服务消息的推送
func (g GameFrame) ServerMessagePush(users []string, data any, session *remote.Session) {
	session.Push(users, data, "ServerMessagePush")
}

// StartGame 游戏开始
func (g GameFrame) StartGame(session *remote.Session, user *proto.RoomUser) {
	// 1. 用户信息变更推送（金币变化） {"gold": 9958, "pushRouter": 'UpdateUserInfoPush'}
	users := g.getAllUsers()
	g.ServerMessagePush(users, UpdateUserInfoPushGold(user.UserInfo.Gold), session)

	// 2. 庄家推送 {"type":414,"data":{"bankerChairID":0},"pushRouter":"GameMessagePush"}
	if g.gameData.CurBureau == 0 {
		g.gameData.BankerChairID = utils.Rand(len(users))
	}
	g.gameData.CurChairID = g.gameData.BankerChairID // 当前有操作的座次号设置为庄家
	g.ServerMessagePush(users, GameBankerPushData(g.gameData.BankerChairID), session)

	// 3. 局数推送{"type":411,"data":{"curBureau":6},"pushRouter":"GameMessagePush"}
	g.gameData.CurBureau++
	g.ServerMessagePush(users, GameBureauPushData(g.gameData.CurBureau), session)

	// 4. 游戏状态推送
	// 第一步：推送发牌，第二步：推送下分，推送用户操作
	g.gameData.GameStatus = SendCards                                                 // 发牌
	g.ServerMessagePush(users, GameStatusPushData(g.gameData.GameStatus, 0), session) //初始发牌不需要倒计时，给0

	// 5. 发牌推送
	g.sendCards(session)

	// 6. 下分推送
	// 推送下分状态
	g.gameData.GameStatus = PourScore // 下分
	g.ServerMessagePush(users, GameStatusPushData(g.gameData.GameStatus, 30), session)
	g.gameData.CurScore = g.gameRule.AddScores[0] * g.gameRule.BaseScore // 当前分数=加注分数*底分
	// 给每个玩家推送
	for _, v := range g.room.GetUsers() {
		g.ServerMessagePush([]string{v.UserInfo.Uid}, GamePourScorePushData(v.ChairID, g.gameData.CurScore, g.gameData.CurScore, 1, 0), session)
	}
	// 7. 轮数推送
	g.gameData.Round = 1
	g.ServerMessagePush(users, GameRoundPushData(g.gameData.Round), session)
	// 8. 操作推送
	for _, v := range g.room.GetUsers() {
		// ChairID是做操作的玩家的座次号， 表示是哪个用户在做操作
		g.ServerMessagePush([]string{v.UserInfo.Uid}, GameTurnPushData(g.gameData.CurChairID, g.gameData.CurScore), session)
	}
}

func (g GameFrame) GameMessageHandle(user *proto.RoomUser, session *remote.Session, msg []byte) {
	//1. 解析参数
	var req MessageReq
	json.Unmarshal(msg, &req)

	//2. 根据不同的类型 触发不同的操作
	// 如果是看牌请求，触发看牌的操作
	if req.Type == GameLookNotify {
		g.onGameLook(user, session, req.Data.Cuopai)
	} else if req.Type == GamePourScoreNotify {
		g.onGamePourScore(user, session, req.Data.Score, req.Data.Type)
	} else if req.Type == GameCompareNotify {
		g.onGameCompare(user, session, req.Data.ChairID)
	} else if req.Type == GameAbandonNotify {
		g.onGameAbandon(user, session)
	} else if req.Type == GameChatNotify {
		g.onGameChat(user, session, req.Data)
	}
}

func (g GameFrame) getAllUsers() []string {
	users := make([]string, 0)
	for _, v := range g.room.GetUsers() {
		users = append(users, v.UserInfo.Uid)
	}
	return users
}

// 发牌动作
func (g GameFrame) sendCards(session *remote.Session) {
	// 1. 洗牌
	g.logic.washCards()
	for i := 0; i < g.gameData.ChairCount; i++ {
		// 如果这个座位号的玩家在游戏
		if g.IsPlayingChairID(i) {
			// 2. 发牌,每个座次发三张牌
			g.gameData.HandCards[i] = g.logic.getCards()
		}
	}
	// 发牌后，推送的时候，如果玩家没有看牌，就是暗牌
	hands := make([][]int, g.gameData.ChairCount)
	for i, v := range g.gameData.HandCards {
		// 判断是否已经发牌
		if v != nil {
			hands[i] = []int{0, 0, 0}
		}
	}
	// 把每个玩家的手牌发给对应的玩家
	g.ServerMessagePush(g.getAllUsers(), GameSendCardsPushData(hands), session)
}

func (g GameFrame) IsPlayingChairID(chairID int) bool {
	for _, v := range g.room.GetUsers() {
		// 如果遍历到的玩家的座位号 == 传进来的座位号，并且这个玩家正在游戏
		if v.ChairID == chairID && v.UserStatus == proto.Playing {
			return true
		}
	}
	return false
}

func (g GameFrame) onGameLook(user *proto.RoomUser, session *remote.Session, cuopai bool) {
	// 判断是当前用户还是其它用户，两种用户推送不同内容
	// 判断玩家状态是否符合条件
	if g.gameData.GameStatus != PourScore || g.gameData.CurChairID != user.ChairID {
		logs.Warn("ID:%s room, 三张 game look err:gameStatus=%d,curChairID=%d,chairID=%d",
			g.room.GetId(), g.gameData.GameStatus, g.gameData.CurChairID, user.ChairID)
		return
	}
	if !g.IsPlayingChairID(user.ChairID) {
		logs.Warn("ID:%s room, 三张 game look err: not playing", g.room.GetId())
		return
	}
	// 该用户的状态更改为 已看牌
	g.gameData.UserStatusArray[user.ChairID] = Look
	g.gameData.LookCards[user.ChairID] = 1
	for _, v := range g.room.GetUsers() {
		if g.gameData.CurChairID == v.ChairID {
			// 如果遍历到的玩家是当前正在操作的玩家， 把看牌的数据推送给操作者
			g.ServerMessagePush([]string{v.UserInfo.Uid},
				GameLookPushData(g.gameData.CurChairID, g.gameData.HandCards[v.ChairID], cuopai),
				session)
		} else {
			// 如果遍历到的玩家是其它玩家
			g.ServerMessagePush([]string{v.UserInfo.Uid},
				GameLookPushData(g.gameData.CurChairID, nil, cuopai),
				session)
		}
	}
}

// 游戏下分相关处理
func (g GameFrame) onGamePourScore(user *proto.RoomUser, session *remote.Session, score int, t int) {
	// 1. 处理下分：保存用户下的分数，推送当前用户下分的信息到客户端
	if g.gameData.GameStatus != PourScore || g.gameData.CurChairID != user.ChairID {
		logs.Warn("ID:%s room, 三张 onGamePourScore err:gameStatus=%d,curChairID=%d,chairID=%d",
			g.room.GetId(), g.gameData.GameStatus, g.gameData.CurChairID, user.ChairID)
		return
	}
	// 确保该玩家在游戏状态（离开或者已输的用户不提供下分操作）
	if !g.IsPlayingChairID(user.ChairID) {
		logs.Warn("ID:%s room, sanzhang onGamePourScore err: not playing",
			g.room.GetId())
		return
	}
	// 确保玩家下的分数>0
	if score < 0 {
		logs.Warn("ID:%s room, sanzhang onGamePourScore err: score lt zero",
			g.room.GetId())
		return
	}
	// 为玩家下的分数做保存,保存在当前轮次的分数中
	if g.gameData.PourScores[user.ChairID] == nil {
		g.gameData.PourScores[user.ChairID] = make([]int, 0)
	}
	g.gameData.PourScores[user.ChairID] = append(g.gameData.PourScores[user.ChairID], score)
	// 计算所有人的分数之和,金池
	scores := 0
	for i := 0; i < g.gameData.ChairCount; i++ {
		if g.gameData.PourScores[i] != nil {
			for _, _score := range g.gameData.PourScores[i] {
				scores += _score
			}
		}
	}
	// 当前座次的总分
	chairCount := 0
	for _, _score := range g.gameData.PourScores[user.ChairID] {
		chairCount += _score
	}
	g.ServerMessagePush(g.getAllUsers(), GamePourScorePushData(user.ChairID, score, chairCount, scores, t), session)
	// 2. 结束下分，座次移动到下一位玩家，推送轮次、游戏状态、操作的座次
	g.endPourScore(session)
}

// 结束下分
func (g GameFrame) endPourScore(session *remote.Session) {
	// 1. 推送轮次 TODO 轮数大于规则的限制 结束游戏 进行结算
	round := g.getCurRound()
	g.ServerMessagePush(g.getAllUsers(), GameRoundPushData(round), session)
	//判断当前的玩家 没有lose的 只剩下一个的时候
	gamerCount := 0
	for i := 0; i < g.gameData.ChairCount; i++ {
		if g.IsPlayingChairID(i) && !utils.Contains(g.gameData.Loser, i) {
			gamerCount++
		}
	}
	// 如果还没输的玩家只有一个，那么这个玩家为胜利者，游戏结束
	if gamerCount == 1 {
		g.startResult(session)
	} else {
		//2. 座次要向前移动一位
		for i := 0; i < g.gameData.ChairCount; i++ {
			g.gameData.CurChairID++
			g.gameData.CurChairID = g.gameData.CurChairID % g.gameData.ChairCount
			if g.IsPlayingChairID(g.gameData.CurChairID) {
				break
			}
		}
		//推送游戏状态
		g.gameData.GameStatus = PourScore
		g.ServerMessagePush(g.getAllUsers(), GameStatusPushData(g.gameData.GameStatus, 30), session)
		//该谁操作了
		g.ServerMessagePush(g.getAllUsers(), GameTurnPushData(g.gameData.CurChairID, g.gameData.CurScore), session)
	}
}

// getCurRound 获取当前轮次
func (g GameFrame) getCurRound() int {
	// 获取当前玩家的座位号
	cur := g.gameData.CurChairID

	// 遍历所有座位
	for i := 0; i < g.gameData.ChairCount; i++ {
		cur++
		cur = cur % g.gameData.ChairCount // 使用模运算确保座位号在有效范围内循环
		// 检查当前座位号的玩家是否正在游戏中
		if g.IsPlayingChairID(cur) {
			return len(g.gameData.PourScores[cur])
		}
	}
	return 1 // 如果没有找到正在游戏中的玩家，默认返回1
}

// 显示游戏结果
func (g GameFrame) startResult(session *remote.Session) {
	//推送 游戏结果状态
	g.gameData.GameStatus = Result
	g.ServerMessagePush(g.getAllUsers(), GameStatusPushData(g.gameData.GameStatus, 0), session)
	if g.gameResult == nil {
		g.gameResult = new(GameResult)
	}
	g.gameResult.Winners = g.gameData.Winner
	g.gameResult.HandCards = g.gameData.HandCards
	g.gameResult.CurScores = g.gameData.CurScores
	g.gameResult.Losers = g.gameData.Loser

	// 计算所赢的分数
	winScores := make([]int, g.gameData.ChairCount)
	for i := range winScores {
		// 检查当前玩家是否有下注记录
		if g.gameData.PourScores[i] != nil {
			scores := 0
			// 累加当前玩家的所有下注分数
			for _, v := range g.gameData.PourScores[i] {
				scores += v
			}
			winScores[i] = -scores // 将当前玩家的总下注分数设置为负数，表示他们失去的分数
			// 将当前玩家的总下注分数平均分配给所有赢家
			for win := range g.gameData.Winner {
				winScores[win] += scores / len(g.gameData.Winner)
			}
		}
	}
	g.gameResult.WinScores = winScores
	g.ServerMessagePush(g.getAllUsers(), GameResultPushData(g.gameResult), session)
	//结算完成 重置游戏 开始下一把
	g.resetGame(session)
	g.gameEnd(session)
}

// 重置游戏
func (gf GameFrame) resetGame(session *remote.Session) {
	g := &GameData{
		GameType:   GameType(gf.gameRule.GameFrameType),
		BaseScore:  gf.gameRule.BaseScore,
		ChairCount: gf.gameRule.MaxPlayerCount,
	}
	g.PourScores = make([][]int, g.ChairCount)
	g.HandCards = make([][]int, g.ChairCount)
	g.LookCards = make([]int, g.ChairCount)
	g.CurScores = make([]int, g.ChairCount)
	g.UserStatusArray = make([]UserStatus, g.ChairCount)
	g.UserTrustArray = []bool{false, false, false, false, false, false, false, false, false, false}
	g.Loser = make([]int, 0)
	g.Winner = make([]int, 0)
	g.GameStatus = GameStatus(None)
	gf.gameData = g
	gf.SendGameStatus(g.GameStatus, 0, session)
	gf.room.EndGame(session)
}

// SendGameStatus 发送游戏状态
func (g GameFrame) SendGameStatus(status GameStatus, tick int, session *remote.Session) {
	g.ServerMessagePush(g.getAllUsers(), GameStatusPushData(status, tick), session)
}

func (g GameFrame) gameEnd(session *remote.Session) {
	//赢家当庄家
	for i := 0; i < g.gameData.ChairCount; i++ {
		if g.gameResult.WinScores[i] > 0 {
			g.gameData.BankerChairID = i
			g.gameData.CurChairID = g.gameData.BankerChairID
		}
	}
	// 5秒钟之后，进入到准备状态
	time.AfterFunc(5*time.Second, func() {
		for _, v := range g.room.GetUsers() {
			g.room.UserReady(v.UserInfo.Uid, session)
		}
	})
}

// 比牌
func (g GameFrame) onGameCompare(user *proto.RoomUser, session *remote.Session, ChairID int) {
	// 1. TODO 先下分，跟注结束后进行比牌
	// 2. 比牌
	fromChairID := user.ChairID
	toChairID := ChairID
	result := g.logic.CompareCards(g.gameData.HandCards[fromChairID], g.gameData.HandCards[toChairID])
	// 3. 处理比牌结果，推送轮数、状态、显示结果等信息
	if result == 0 {
		//主动比牌者 如果是和 主动比牌者输
		result = -1
	}
	winChairID := -1
	loseChairID := -1
	if result > 0 {
		g.ServerMessagePush(g.getAllUsers(), GameComparePushData(fromChairID, toChairID, fromChairID, toChairID), session)
		winChairID = fromChairID
		loseChairID = toChairID
	} else if result < 0 {
		g.ServerMessagePush(g.getAllUsers(), GameComparePushData(fromChairID, toChairID, toChairID, fromChairID), session)
		winChairID = toChairID
		loseChairID = fromChairID
	}
	if winChairID != -1 && loseChairID != -1 {
		g.gameData.UserStatusArray[winChairID] = Win
		g.gameData.UserStatusArray[loseChairID] = Lose
		g.gameData.Loser = append(g.gameData.Loser, loseChairID)
		g.gameData.Winner = append(g.gameData.Winner, winChairID)
	}
	// TODO 赢了之后，需要继续和其它人进行比牌
	if winChairID == fromChairID {

	}
	g.endPourScore(session)
}

// 回应用户弃牌的操作
func (g GameFrame) onGameAbandon(user *proto.RoomUser, session *remote.Session) {
	// 检查用户是否在游玩
	if !g.IsPlayingChairID(user.ChairID) {
		return
	}
	// 检查用户是否落败
	if utils.Contains(g.gameData.Loser, user.ChairID) {
		return
	}
	// 弃牌导致除了棋牌者，其它玩家都是赢家
	g.gameData.Loser = append(g.gameData.Loser, user.ChairID)
	for i := 0; i < g.gameData.ChairCount; i++ {
		if g.IsPlayingChairID(i) && i != user.ChairID {
			g.gameData.Winner = append(g.gameData.Winner, i)
		}
	}
	g.gameData.UserStatusArray[user.ChairID] = Abandon
	//推送弃牌的消息给客户端
	g.send(GameAbandonPushData(user.ChairID, g.gameData.UserStatusArray[user.ChairID]), session)

	// 等一秒之后执行结束下分
	time.AfterFunc(time.Second, func() {
		g.endPourScore(session)
	})
}

func (g GameFrame) send(data any, session *remote.Session) {
	g.ServerMessagePush(g.getAllUsers(), data, session)
}

// 聊天
func (g GameFrame) onGameChat(user *proto.RoomUser, session *remote.Session, data MessageData) {
	g.send(GameChatPushData(user.ChairID, data.Type, data.Msg, data.RecipientID), session)
}

func NewGameFrame(rule proto.GameRule, r base.RoomFrame) *GameFrame {
	gameData := initGameData(rule)
	return &GameFrame{
		room:     r,
		gameRule: rule,
		gameData: gameData,
		logic:    NewLogic(),
	}
}

func initGameData(rule proto.GameRule) *GameData {
	// 初始化 GameData 结构体并设置基本属性
	g := &GameData{
		GameType:   GameType(rule.GameFrameType), // 设置游戏类型
		BaseScore:  rule.BaseScore,               // 设置基础分数
		ChairCount: rule.MaxPlayerCount,          // 设置玩家数量（座次）
	}
	g.PourScores = make([][]int, g.ChairCount)           // 初始化每个玩家的下注分数数组
	g.HandCards = make([][]int, g.ChairCount)            // 初始化每个玩家的手牌数组
	g.LookCards = make([]int, g.ChairCount)              // 初始化每个玩家的看牌状态数组
	g.CurScores = make([]int, g.ChairCount)              // 初始化每个玩家的当前分数数组
	g.UserStatusArray = make([]UserStatus, g.ChairCount) // 初始化每个玩家的状态数组

	// 初始化玩家的托管状态数组，默认所有玩家都不在托管状态
	g.UserTrustArray = []bool{false, false, false, false, false, false, false, false, false, false}
	g.Loser = make([]int, 0)  // 初始化失败者列表，初始为空
	g.Winner = make([]int, 0) // 初始化胜利者列表，初始为空
	return g
}
