package handler

import (
	"common"
	"common/biz"
	"common/config"
	"common/jwts"
	"common/logs"
	"connector/models/request"
	"context"
	"core/repo"
	"core/service"
	"encoding/json"
	"framework/game"
	"framework/net"
)

// EntryHandler 结构体定义了一个处理入口的处理器
type EntryHandler struct {
	userService *service.UserService
}

// Entry 方法处理进入的会话和消息体
func (h *EntryHandler) Entry(session *net.Session, body []byte) (any, error) {
	logs.Info("=================Entry Start============================")
	logs.Info("Entry Handler Entry %v", string(body))
	logs.Info("=================Entry End  ============================")
	var req request.EntryReq
	err := json.Unmarshal(body, &req)
	if err != nil {
		return common.Fail(biz.RequestDataError), nil
	}

	// 校验Token, 解析出来一个uid
	uid, err := jwts.ParseToken(req.Token, config.Conf.Jwt.Secret)
	if err != nil {
		logs.Error("parse token err %v", err)
		return common.Fail(biz.TokenInfoError), nil
	}

	// 根据uid在mongo中查询用户，如果用户不存在，生成一个用户
	user, err := h.userService.FindUserByUid(context.TODO(), uid, req.UserInfo)
	if err != nil {
		return common.Fail(biz.SqlError), nil
	}
	session.Uid = uid
	return common.S(map[string]any{
		"userInfo": user,
		"config":   game.Conf.GetFromGameConfig(),
	}), nil
}

// NewEntryHandler 创建并返回一个新的 EntryHandler 实例
func NewEntryHandler(r *repo.Manager) *EntryHandler {
	return &EntryHandler{
		userService: service.NewUserService(r),
	}
}
