package response

import (
	"common"
	"hall/models/request"
)

// UpdateUserAddressRes 更新玩家地址的请求 的 响应
type UpdateUserAddressRes struct {
	common.Result
	UpdateUserData request.UpdateUserAddressReq `json:"updateUserData"`
}
