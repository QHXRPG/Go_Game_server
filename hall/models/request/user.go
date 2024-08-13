package request

// UpdateUserAddressReq 更新玩家地址
type UpdateUserAddressReq struct {
	Address  string `json:"address,omitempty"`
	Location string `json:"location,omitempty"`
}
