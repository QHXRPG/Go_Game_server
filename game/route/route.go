package route

import (
	"common/logs"
	"core/repo"
	"framework/node"
	"game/handler"
	"game/logic"
)

// Register 函数注册所有的处理器并返回一个 LoginHandler
func Register(r *repo.Manager) node.LogicHandler {
	handlers := make(node.LogicHandler)
	unionManager := logic.NewUnionManager()

	// 将 unionHandler 的 createRoom 方法注册到 handlers 中
	unionHandler := handler.NewUnionHandler(r, unionManager) // 创建一个新的 unionHandler 实例
	handlers["unionHandler.createRoom"] = unionHandler.CreateRoom
	logs.Info("register handlers userHandler.updateUserAddress")

	// 将 unionHandler 的 joinRoom 方法注册到 handlers 中
	handlers["unionHandler.joinRoom"] = unionHandler.JoinRoom
	logs.Info("register handlers unionHandler.joinRoom")

	// 将 gameHandler 的 roomMessageNotify 方法注册到 handlers 中
	gameHandler := handler.NewGameHandler(r, unionManager) // 创建一个新的 gameHandler 实例
	handlers["gameHandler.roomMessageNotify"] = gameHandler.RoomMessageNotify
	logs.Info("register handlers userHandler.updateUserAddress")

	// 将 gameHandler 的 gameMessageNotify 方法注册到 handlers 中
	handlers["gameHandler.gameMessageNotify"] = gameHandler.GameMessageNotify
	logs.Info("register handlers userHandler.updateUserAddress")

	return handlers
}
