package handler

import (
	"common"
	"common/biz"
	"common/logs"
	"core/repo"
	"core/service"
	"encoding/json"
	"framework/remote"
	"hall/models/request"
	"hall/models/response"
)

type UserHandler struct {
	userService *service.UserService
}

// UpdateUserAddress 更新玩家地址
func (h *UserHandler) UpdateUserAddress(session *remote.Session, msg []byte) any {
	logs.Info("UpdateUserAddress msg:%v", string(msg))
	var req request.UpdateUserAddressReq // 先给定一个结构体格式
	err := json.Unmarshal(msg, &req)     // Unmarshal() 按照req结构体来反序列化，最后将msg赋值给req
	if err != nil {
		return common.Fail(biz.RequestDataError)
	}
	err = h.userService.UpdateUserAddressByUid(session.GetUid(), req)
	if err != nil {
		logs.Error("UserHandler.UpdateUserAddress err:%v", err)
		return nil
	}
	res := response.UpdateUserAddressRes{}
	res.Code = biz.OK
	res.UpdateUserData = req
	return res
}

func NewUserHandler(r *repo.Manager) *UserHandler {
	return &UserHandler{
		userService: service.NewUserService(r),
	}
}
