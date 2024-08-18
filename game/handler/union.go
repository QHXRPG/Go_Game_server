package handler

import (
	"common"
	"common/biz"
	"context"
	"core/repo"
	"core/service"
	"encoding/json"
	"framework/remote"
	"game/logic"
	"game/models/request"
)

type UnionHandler struct {
	unionManager *logic.UnionManager
	userService  *service.UserService
}

func (h *UnionHandler) CreateRoom(session *remote.Session, msg []byte) any {
	// union 联盟持有房间
	// unionManager 管理联盟
	// room房间关联game接口，实现多个不同的游戏
	// 1. 接收参数
	uid := session.GetUid()
	if len(uid) <= 0 {
		return common.Fail(biz.InvalidUsers)
	}
	var req request.CreateRomRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return common.Fail(biz.RequestDataError)
	}

	// 2. 根据session用户id 查询用户信息
	userData, err := h.userService.FindUserByUid(context.TODO(), uid)
	if err != nil {
		return common.Fail(err)
	}
	if userData == nil {
		return common.Fail(biz.InvalidUsers)
	}

	// 3. 根据游戏规则、游戏类型、用户信息(创建房间的用户) 创建房间
	//TODO 需要判断session中是否有roomId，代表此用户已在房间中，不能再次创建房间中
	union := h.unionManager.GetUnion(req.UnionID)
	err = union.CreateRoom(h.userService, session, req, userData)
	if err != nil {
		return common.Fail(err)
	}
	return common.S(nil)
}

// JoinRoom 用户输入房间号加入房间,流程与 CreateRoom 相似
func (h *UnionHandler) JoinRoom(session *remote.Session, msg []byte) any {
	// 1. 接收参数
	uid := session.GetUid()
	if len(uid) <= 0 {
		return common.Fail(biz.InvalidUsers)
	}
	var req request.JoinRoomReq
	if err := json.Unmarshal(msg, &req); err != nil {
		return common.Fail(biz.RequestDataError)
	}
	// 2. 根据session用户id 查询用户信息
	userData, err := h.userService.FindUserByUid(context.TODO(), uid)
	if err != nil {
		return common.Fail(err)
	}
	if userData == nil {
		return common.Fail(biz.InvalidUsers)
	}
	// 3. 加入房间
	bizErr := h.unionManager.JoinRoom(session, req.RoomID, userData)
	if bizErr != nil {
		return common.Fail(bizErr)
	}
	return common.S(nil)
}

func NewUnionHandler(r *repo.Manager, um *logic.UnionManager) *UnionHandler {
	return &UnionHandler{
		unionManager: um,
		userService:  service.NewUserService(r),
	}
}
