package connector

// connector 用于管理客户端和服务器之间通信的组件：处理网络连接、消息传递和路由管理

import (
	"common/logs"
	"fmt"
	"framework/game"
	"framework/net"
	"framework/remote"
)

// Connector 结构体，包括是否运行中的状态、WebSocket管理器和处理器
type Connector struct {
	isRunning        bool
	websocketManager *net.Manager
	handlers         net.LogicHandler
	remoteClient     remote.Client
}

// Default 函数返回一个默认的Connector实例
func Default() *Connector {
	return &Connector{
		handlers: make(net.LogicHandler),
	}
}

// Run 方法启动Connector服务
func (c *Connector) Run(serverId string) {
	if !c.isRunning {
		// 启动WebSocket和NATS
		c.websocketManager = net.NewManager()
		c.websocketManager.ConnectorHandlers = c.handlers
		// 启动nats nats server不会存储消息
		c.remoteClient = remote.NewNatsClient(serverId, c.websocketManager.RemoteReadChan)
		c.remoteClient.Run()
		c.websocketManager.RemoteClient = c.remoteClient
		c.Serve(serverId)
	}
}

// Close 方法关闭Connector服务
func (c *Connector) Close() {
	if c.isRunning {
		// 关闭WebSocket和NATS
		c.websocketManager.Close()
		c.isRunning = false
	}
}

// Serve 方法配置并运行websocketManager
func (c *Connector) Serve(serverId string) {
	logs.Info("run connector server id:%s", serverId)
	// 地址，读取配置文件，在游戏中可能加载很多信息（配置），如果写到yml文件中会比较复杂，不容易维护
	// 游戏中的配置读取，一般采取json的方式，读取json配置文件
	c.websocketManager.ServerId = serverId
	connectorConfig := game.Conf.GetConnector(serverId)
	if connectorConfig == nil {
		logs.Fatal("not found connector config")
		return
	}
	addr := fmt.Sprintf("%s:%d", connectorConfig.Host, connectorConfig.ClientPort)
	c.isRunning = true
	err := c.websocketManager.Run(addr)
	if err != nil {
		return
	}
}

// RegisterHandler 方法注册处理器
func (c *Connector) RegisterHandler(handlers net.LogicHandler) {
	c.handlers = handlers
}
