package route

import (
	"connector/handler"
	"core/repo"
	"framework/net"
)

// Register 函数注册所有的处理器并返回一个 LogicHandler
func Register(r *repo.Manager) net.LogicHandler {
	handlers := make(net.LogicHandler)
	entryHandler := handler.NewEntryHandler(r) // 创建一个新的 EntryHandler 实例

	// 将 EntryHandler 的 Entry 方法注册到 handlers 中
	handlers["entryHandler.entry"] = entryHandler.Entry
	return handlers
}
