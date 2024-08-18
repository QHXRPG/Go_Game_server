package logic

import (
	"common/biz"
	"core/models/entity"
	"fmt"
	"framework/msError"
	"framework/remote"
	"game/compone/room"
	"math/rand"
	"sync"
	"time"
)

type UnionManager struct {
	sync.RWMutex
	unionList map[int64]*Union
}

func NewUnionManager() *UnionManager {
	return &UnionManager{
		unionList: make(map[int64]*Union),
	}
}

// GetUnion 拿到Union
func (u *UnionManager) GetUnion(unionId int64) *Union {
	u.Lock()
	u.Unlock()
	union, ok := u.unionList[unionId]
	if ok {
		return union
	}
	union = NewUnion(u)
	u.unionList[unionId] = union
	return union
}

// CreateRoomId 创建房间ID
func (u *UnionManager) CreateRoomId() string {
	// 随机数去创建
	roomID := u.genRoomId()
	for _, v := range u.unionList {
		_, ok := v.RoomList[roomID]
		if ok {
			return u.CreateRoomId()
		}
	}
	return roomID
}

// genRoomId 生成房间号
func (u *UnionManager) genRoomId() string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	// 房间号是6位数
	roomIdInt := rand.Int63n(999999)
	if roomIdInt < 100000 {
		roomIdInt += 100000
	}
	return fmt.Sprintf("%d", roomIdInt)
}

func (u *UnionManager) GetRoomById(roomId string) *room.Room {
	for _, v := range u.unionList {
		r, ok := v.RoomList[roomId]
		if ok {
			return r
		}
	}
	return nil
}

// JoinRoom 用户加入房间
func (u *UnionManager) JoinRoom(session *remote.Session, roomId string, data *entity.User) *msError.Error {
	for _, v := range u.unionList {
		r, ok := v.RoomList[roomId] // 通过玩家输入的房间号找到Room
		if ok {
			return r.JoinRoom(session, data)
		}
	}
	return biz.RoomNotExist
}
