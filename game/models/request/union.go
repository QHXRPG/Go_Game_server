package request

import "game/compone/proto"

type CreateRomRequest struct {
	UnionID    int64          `json:"unionID"`
	GameRuleID string         `json:"gameRuleID"`
	GameRule   proto.GameRule `json:"gameRule"`
}

// JoinRoomReq 加入房间的请求
type JoinRoomReq struct {
	RoomID string `json:"roomID"`
}
