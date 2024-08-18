package logic

import (
	"core/models/entity"
	"core/service"
	"framework/msError"
	"framework/remote"
	"game/compone/room"
	"game/models/request"
	"sync"
)

type Union struct {
	sync.RWMutex
	Id           int64
	unionManager *UnionManager
	RoomList     map[string]*room.Room
}

// DismissRoom 解散房间
func (u *Union) DismissRoom(roomId string) {
	u.Lock()
	defer u.Unlock()
	delete(u.RoomList, roomId)
}

func (u *Union) CreateRoom(service *service.UserService, session *remote.Session, req request.CreateRomRequest, userData *entity.User) *msError.Error {
	// 1. 需要创建一个房间，生成一个房间号
	roomId := u.unionManager.CreateRoomId()
	newRoom := room.NewRoom(roomId, req.UnionID, req.GameRule, u)
	u.RoomList[roomId] = newRoom

	// 创建房间后进入房间
	return newRoom.UserEntryRoom(session, userData)
}

func NewUnion(m *UnionManager) *Union {
	return &Union{
		RoomList:     make(map[string]*room.Room),
		unionManager: m,
	}
}
