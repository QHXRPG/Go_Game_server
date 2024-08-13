package route

import (
	"common/logs"
	"core/repo"
	"framework/node"
	"hall/handler"
)

// Register 函数注册所有的处理器并返回一个 LoginHandler
func Register(r *repo.Manager) node.LogicHandler {
	handlers := make(node.LogicHandler)
	userHandler := handler.NewUserHandler(r) // 创建一个新的 NewUserHandler 实例

	// 将 userHandler 的 updateUserAddress 方法注册到 handlers 中
	handlers["userHandler.updateUserAddress"] = userHandler.UpdateUserAddress
	logs.Info("register handlers userHandler.updateUserAddress")
	return handlers
}

// userHandler.updateUserAddress
// userHandler.updateUserAddress
