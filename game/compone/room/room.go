package room

import (
	"common/logs"
	"core/models/entity"
	"framework/msError"
	"framework/remote"
	"game/compone/base"
	"game/compone/proto"
	"game/compone/sz"
	"game/models/request"
	"sync"
	"time"
)

type Room struct {
	sync.RWMutex
	Id            string
	unionID       int64
	gameRule      proto.GameRule
	users         map[string]*proto.RoomUser // 房间里面的用户
	RoomCreator   *proto.RoomCreator         // 房间创建者
	GameFrame     GameFrame                  // 具体的游戏实例，将游戏逻辑与房间管理逻辑分离
	kickSchedules map[string]*time.Timer
	roomDismissed bool
	union         base.UnionBase
	gameStarted   bool
}

func (r *Room) GetId() string {
	return r.Id
}

// UserEntryRoom 用户进入房间
func (r *Room) UserEntryRoom(session *remote.Session, data *entity.User) *msError.Error {
	r.RoomCreator = &proto.RoomCreator{
		Uid: data.Uid,
	}
	if r.unionID == 1 {
		r.RoomCreator.CreatorType = proto.UserCreatorType
	} else {
		r.RoomCreator.CreatorType = proto.UnionCreatorType
	}
	// 分配座位号，最多6人参加 分配0-56个号
	chairID := r.getEmptyChairID()
	_, ok := r.users[data.Uid]
	if !ok {
		r.users[data.Uid] = proto.ToRoomUser(data, chairID)
	}
	// 2. 将房间号推送给客户端,更新数据库，将当前房间号存储起来
	r.UpdateUserInfoRoomPush(session, data.Uid)
	session.Put("roomId", r.Id)
	// 3. 将游戏类型推送给客户端（用户进入游戏的推送）
	r.SelfEntryRoomPush(session, data.Uid)
	// 4. 通知其它玩家该用户进入房间
	r.OtherUserEntryRoomPush(session, data.Uid)
	go r.addKickScheduleEvent(session, data.Uid)
	return nil
}

//********** game服务是一个node节点，因此game其实是一个nats的客户端 ***********************************************/
//********** 所以现在要将消息发送给 connector，connector 将消息发给客户端，connector是一个websocket的连接（双向通道）***//

func (r *Room) UpdateUserInfoRoomPush(session *remote.Session, uid string) {
	// {roomID: '336842', pushRouter: 'UpdateUserInfoPush'}
	pushMsg := map[string]any{
		"roomID":     r.Id,
		"pushRouter": "UpdateUserInfoPush",
	}
	session.Push([]string{uid}, pushMsg, "ServerMessagePush")
}

// SelfEntryRoomPush 用户进入房间的消息推送，推送给进入房间的客户端
func (r *Room) SelfEntryRoomPush(session *remote.Session, uid string) {
	// {gameType: 1, pushRouter: 'SelfEntryRoomPush'}
	pushMsg := map[string]any{
		"gameType":   r.gameRule.GameType,
		"pushRouter": "SelfEntryRoomPush",
	}
	session.Push([]string{uid}, pushMsg, "ServerMessagePush")
}

func (r *Room) RoomMessageHandle(session *remote.Session, req request.RoomMessageReq) {
	//  处理用户准备的Notify
	if req.Type == proto.UserReadyNotify {
		r.userReady(session.GetUid(), session)
	}
	// 处理客户端请求房间场景的Notify
	if req.Type == proto.GetRoomSceneInfoNotify {
		r.getRoomSceneInfoPush(session)
	}
}

// 推送房间场景信息
func (r *Room) getRoomSceneInfoPush(session *remote.Session) {
	userInfoArr := make([]*proto.RoomUser, 0)
	for _, user := range r.users {
		userInfoArr = append(userInfoArr, user)
	}
	data := map[string]any{
		"type":       proto.GetRoomSceneInfoPush,
		"pushRouter": "RoomMessagePush",
		"data": map[string]any{
			"roomID":          r.Id,
			"roomCreatorInfo": r.RoomCreator,
			"gameRule":        r.gameRule,
			"roomUserInfoArr": userInfoArr, // 所有用户的信息
			"gameData":        r.GameFrame.GetGameData(session),
		},
	}
	session.Push([]string{session.GetUid()}, data, "ServerMessagePush")
}

// 添加用户长时间未准备踢出的任务
func (r *Room) addKickScheduleEvent(session *remote.Session, uid string) {
	r.Lock()
	defer r.Unlock()
	t, ok := r.kickSchedules[uid]
	if ok {
		t.Stop()
		delete(r.kickSchedules, uid)
	}

	// 添加定时任务，30秒后执行
	r.kickSchedules[uid] = time.AfterFunc(30*time.Second, func() {
		logs.Info("kick 定时执行，代表 用户长时间未准备,uid=%v", uid)
		//取消定时任务
		timer, ok := r.kickSchedules[uid]
		if ok {
			timer.Stop()
		}
		delete(r.kickSchedules, uid) //删除
		//需要判断用户是否该踢出
		user, ok := r.users[uid]
		if ok {
			if user.UserStatus < proto.Ready {
				r.kickUser(user, session)
				//踢出房间之后，需要判断是否可以解散房间
				if len(r.users) == 0 {
					r.dismissRoom()
				}
			}
		}
	})
}

func (r *Room) ServerMessagePush(users []string, data any, session *remote.Session) {
	session.Push(users, data, "ServerMessagePush")
}

// 踢出用户
func (r *Room) kickUser(user *proto.RoomUser, session *remote.Session) {
	//将roomId设为空，并将这消息发给当前用户，意味着踢出该用户
	r.ServerMessagePush([]string{user.UserInfo.Uid}, proto.UpdateUserInfoPush(""), session)
	//通知其他人该用户离开房间
	users := make([]string, 0)
	for _, v := range r.users {
		users = append(users, v.UserInfo.Uid) // 收集其余玩家并存入user
	}
	r.ServerMessagePush(users, proto.UserLeaveRoomPushData(user), session)
	delete(r.users, user.UserInfo.Uid) // 将提出的用户删除
}

// 解散房间
func (r *Room) dismissRoom() {
	r.Lock()
	defer r.Unlock()
	if r.roomDismissed {
		return
	}
	r.roomDismissed = true
	//解散 删除union当中存储的room信息
	r.cancelAllScheduler() // 取消房间内所有的定时任务
	r.union.DismissRoom(r.Id)
}

func (r *Room) cancelAllScheduler() {
	//需要将房间所有的任务都取消掉
	for uid, v := range r.kickSchedules {
		v.Stop()
		delete(r.kickSchedules, uid)
	}
}

func (r *Room) UserReady(uid string, session *remote.Session) {
	r.userReady(uid, session)
}

func (r *Room) getEmptyChairID() int {
	if len(r.users) == 0 {
		return 0
	}
	r.Lock()
	defer r.Unlock()
	chairID := 0
	for _, v := range r.users {
		if v.ChairID == chairID {
			//座位号被占用了
			chairID++ // 移到下一个座位
		}
	}
	return chairID
}

// 用户准备
func (r *Room) userReady(uid string, session *remote.Session) {
	//1. push用户的座次,修改用户的状态，取消定时任务
	user, ok := r.users[uid] // 通过uid拿到玩家
	if !ok {
		return
	}
	user.UserStatus = proto.Ready
	// 取消定时任务
	timer, ok := r.kickSchedules[uid]
	if ok {
		timer.Stop()
		delete(r.kickSchedules, uid)
	}
	// 给所有用户推送
	allUsers := r.AllUsers()
	r.ServerMessagePush(allUsers, proto.UserReadyPushData(user.ChairID), session)
	// 2. 准备好之后，判断是否需要开始游戏
	if r.IsStartGame() {
		r.startGame(session, user)
	}
}

// AllUsers 获取当前房间下所有玩家
func (r *Room) AllUsers() []string {
	users := make([]string, 0)
	for _, v := range r.users {
		users = append(users, v.UserInfo.Uid)
	}
	return users
}

func (r *Room) IsStartGame() bool {
	//房间内准备的人数 >= 最小开始游戏人数
	userReadyCount := 0
	for _, v := range r.users {
		if v.UserStatus == proto.Ready {
			userReadyCount++ // 统计已准备的玩家的数量
		}
	}
	// 条件一：房间内的人数=已准备的人数；  条件二：已准备的人数>=最小开始游戏人数
	if len(r.users) == userReadyCount && userReadyCount >= r.gameRule.MinPlayerCount {
		return true
	}
	return false
}

func (r *Room) startGame(session *remote.Session, user *proto.RoomUser) {
	if r.gameStarted {
		return
	}
	r.gameStarted = true
	// 设置房间里面的玩家的状态为 正在游戏中
	for _, v := range r.users {
		v.UserStatus = proto.Playing
	}
	r.GameFrame.StartGame(session, user)
}

func (r *Room) JoinRoom(session *remote.Session, data *entity.User) *msError.Error {

	return r.UserEntryRoom(session, data)
}

// OtherUserEntryRoomPush 对其它玩家关于进入房间玩家的消息的推送
func (r *Room) OtherUserEntryRoomPush(session *remote.Session, uid string) {
	others := make([]string, 0)
	for _, v := range r.users {
		if v.UserInfo.Uid == uid {
			others = append(others, v.UserInfo.Uid)
		}
	}
	user, ok := r.users[uid]
	if ok {
		r.ServerMessagePush(others, proto.OtherUserEntryRoomPushData(user), session)
	}
}

func NewRoom(id string, unionID int64, rule proto.GameRule, u base.UnionBase) *Room {
	room := &Room{
		Id:            id,
		unionID:       unionID,
		gameRule:      rule,
		users:         make(map[string]*proto.RoomUser),
		kickSchedules: make(map[string]*time.Timer),
		union:         u,
	}
	if rule.GameType == int(proto.PinSanZhang) {
		room.GameFrame = sz.NewGameFrame(rule, room)
	}
	return room
}

func (r *Room) GetUsers() map[string]*proto.RoomUser {
	return r.users
}

// EndGame 结束游戏
func (r *Room) EndGame(session *remote.Session) {
	r.gameStarted = false
	// 把所有玩家的状态设置为初始化的状态
	for k := range r.users {
		r.users[k].UserStatus = proto.None
	}
}

// GameMessageHandle  游戏消息处理
func (r *Room) GameMessageHandle(session *remote.Session, msg []byte) {
	//需要游戏去处理具体的消息
	user, ok := r.users[session.GetUid()]
	if !ok {
		return
	}
	r.GameFrame.GameMessageHandle(user, session, msg)
}
