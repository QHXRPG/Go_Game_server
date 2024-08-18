package handler

import (
	"common"
	"common/biz"
	"core/repo"
	"core/service"
	"encoding/json"
	"fmt"
	"framework/remote"
	"game/logic"
	"game/models/request"
)

type GameHandler struct {
	um          *logic.UnionManager
	userService *service.UserService
}

// RoomMessageNotify 处理客户端发来的关于房间的相关提示消息（Notify级别消息，只处理不回应）
func (h *GameHandler) RoomMessageNotify(session *remote.Session, msg []byte) any {

	if len(session.GetUid()) <= 0 {
		return common.Fail(biz.InvalidUsers)
	}
	// 把获取房间信息的请求反序列化为req
	var req request.RoomMessageReq
	if err := json.Unmarshal(msg, &req); err != nil {
		return common.Fail(biz.RequestDataError)
	}
	// room去处理这一块的业务
	rommId, ok := session.Get("roomId")
	if !ok {
		return common.Fail(biz.NotInRoom)
	}
	room := h.um.GetRoomById(fmt.Sprintf("%v", rommId))
	if room == nil {
		return common.Fail(biz.RoomNotExist)
	}
	room.RoomMessageHandle(session, req) // 房间消息处理
	return nil
}

// GameMessageNotify 处理用户的看牌请求
func (h *GameHandler) GameMessageNotify(session *remote.Session, msg []byte) any {
	if len(session.GetUid()) <= 0 {
		return common.Fail(biz.InvalidUsers)
	}
	//room去处理这块的业务
	roomId, ok := session.Get("roomId")
	if !ok {
		return common.Fail(biz.NotInRoom)
	}
	roomManager := h.um.GetRoomById(fmt.Sprintf("%v", roomId))
	if roomManager == nil {
		return common.Fail(biz.NotInRoom)
	}
	roomManager.GameMessageHandle(session, msg)
	return nil
}

func NewGameHandler(r *repo.Manager, um *logic.UnionManager) *GameHandler {
	return &GameHandler{
		um:          um,
		userService: service.NewUserService(r),
	}
}
