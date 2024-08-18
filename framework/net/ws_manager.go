package net

import (
	"common/logs"
	"common/utils"
	"encoding/json"
	"errors"
	"fmt"
	"framework/game"
	"framework/protocol"
	"framework/remote"
	"github.com/gorilla/websocket"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// 这个文件定义了一个 `Manager` 结构体，⭐用于管理 WebSocket 连接和处理客户端消息。
// 包含了启动 HTTP 服务器、处理 WebSocket 连接升级、管理客户端连接、解码和路由消息包、
// 以及处理特定类型的消息（如握手、心跳、数据和踢出）的功能。
// 通过 `Manager` 可以高效地管理和处理多个客户端的 WebSocket 连接。

var (
	// websocketUpgrade 是一个全局的 websocket.Upgrader 实例，用于升级 HTTP 连接到 WebSocket 连接
	websocketUpgrade = websocket.Upgrader{
		// CheckOrigin 是一个函数，用于检查请求的来源。这里返回 true，表示接受所有来源的请求
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

// CheckOriginHandler 是一个函数类型，用于自定义检查请求来源的逻辑
type CheckOriginHandler func(r *http.Request) bool

// Manager 结构体管理 WebSocket 连接
type Manager struct {
	sync.RWMutex                             // 读写锁，用于保护共享资源
	websocketUpgrade   *websocket.Upgrader   // WebSocket 升级器
	CheckOriginHandler CheckOriginHandler    // 自定义检查请求来源的处理函数
	clients            map[string]Connection // 存储客户端连接
	ServerId           string
	ClientReadChan     chan *MsgPack
	handlers           map[protocol.PackageType]EventHandler // packet处理器
	ConnectorHandlers  LogicHandler
	RemoteReadChan     chan []byte
	RemoteClient       remote.Client
	RemotePushChan     chan *remote.Msg
}

// HandlerFunc 定义处理函数类型
type HandlerFunc func(session *Session, body []byte) (any, error)

// LogicHandler 是处理函数的映射
type LogicHandler map[string]HandlerFunc

// EventHandler 定义事件处理函数类型
type EventHandler func(packet *protocol.Packet, c Connection) error

// Run 启动 HTTP 服务器并监听游戏前端的连接
func (m *Manager) Run(addr string) error {
	go m.clientReadChanHandler()
	go m.remoteReadChanHandler()
	go m.RemotePushChanHandler()
	http.HandleFunc("/", m.serveWS) // 将 HTTP 请求交给 m.serveWS 函数处理
	// 设置不同的消息处理器
	m.setupEventHandlers()
	err := http.ListenAndServe(addr, nil) // 启动 HTTP 服务器
	if err != nil {
		logs.Fatal("ListenAndServe: ", err)
		return err
	}
	return nil
}

// serveWS 来一个请求，生成一个客户端
func (m *Manager) serveWS(writer http.ResponseWriter, request *http.Request) {
	// 如果 websocketUpgrade 未初始化，则使用全局的 websocketUpgrade
	if m.websocketUpgrade == nil {
		m.websocketUpgrade = &websocketUpgrade
	}
	// 升级 HTTP 连接到 WebSocket 连接
	wsConn, err := m.websocketUpgrade.Upgrade(writer, request, nil)
	if err != nil {
		logs.Error("WebSocket upgrade failed: ", err)
		return
	}
	client := NewWsConnection(wsConn, m) // 每来一个客户端都会生成一个client
	m.addClient(client)
	client.Run()
}

// addClient 将新连接的客户端添加到管理器
func (m *Manager) addClient(client *WsConnection) {
	m.Lock()
	defer m.Unlock()
	m.clients[client.Cid] = client
}

// removeClient 从管理器中移除客户端
func (m *Manager) removeClient(wc WsConnection) {
	for cid, c := range m.clients {
		if cid == wc.Cid {
			c.Close()
			delete(m.clients, cid)
		}
	}
}

// clientReadChanHandler 处理来自客户端的消息
func (m *Manager) clientReadChanHandler() {
	for {
		select {
		case body, ok := <-m.ClientReadChan:
			if ok {
				m.decodeClientPack(body)
			}
		}
	}
}

// decodeClientPack 解码客户端消息包
func (m *Manager) decodeClientPack(body *MsgPack) {
	// 解析协议
	packet, err := protocol.Decode(body.Body)
	if err != nil {
		logs.Error("Decode failed: ", err)
		return
	}
	if err := m.routeEvent(packet, body.Cid); err != nil {
		logs.Error("route event failed: ", err)
	}
}

// Close 关闭所有客户端连接
func (m *Manager) Close() {
	for cid, v := range m.clients {
		v.Close()
		delete(m.clients, cid)
	}
}

// routeEvent 根据 packet 的类型路由事件
func (m *Manager) routeEvent(packet *protocol.Packet, cid string) error {
	// 根据 packet 的类型做不同的处理
	conn, ok := m.clients[cid]
	if ok {
		handler, ok := m.handlers[packet.Type]
		if ok {
			return handler(packet, conn)
		}
		return errors.New("not found packet handler")
	}
	return errors.New("not found client")
}

// setupEventHandlers 设置不同类型的事件处理器
func (m *Manager) setupEventHandlers() {
	m.handlers[protocol.Handshake] = m.HandshakeHandler
	m.handlers[protocol.HandshakeAck] = m.HandshakeAckHandler
	m.handlers[protocol.Heartbeat] = m.HeartbeatHandler
	m.handlers[protocol.Data] = m.MessageHandler
	m.handlers[protocol.Kick] = m.KickHandler
}

// HandshakeHandler 处理握手消息
func (m *Manager) HandshakeHandler(packet *protocol.Packet, c Connection) error {
	response := protocol.HandshakeResponse{
		Code: 200,
		Sys: protocol.Sys{
			Heartbeat: 3,
		},
	}
	data, _ := json.Marshal(response)
	buf, err := protocol.Encode(packet.Type, data)
	if err != nil {
		logs.Error("Encode failed: ", err)
		return err
	}
	return c.SendMessage(buf)
}

// HandshakeAckHandler 处理握手确认消息
func (m *Manager) HandshakeAckHandler(packet *protocol.Packet, c Connection) error {
	logs.Info("receiver handshake ack handler")
	return nil
}

// HeartbeatHandler 处理心跳消息
func (m *Manager) HeartbeatHandler(packet *protocol.Packet, c Connection) error {
	logs.Info("receiver heartbeat handler:%v", packet.Type)
	var response []byte
	data, _ := json.Marshal(response)
	buf, err := protocol.Encode(packet.Type, data)
	if err != nil {
		logs.Error("Encode failed: ", err)
		return err
	}
	return c.SendMessage(buf)
}

// MessageHandler 处理数据消息
func (m *Manager) MessageHandler(packet *protocol.Packet, c Connection) error {
	// 获取消息体
	message := packet.MessageBody()
	logs.Info("receiver message handler, type=%v, router:%v, data:%v", message.Type, message.Route, string(message.Data))

	// 解析路由
	routeStr := message.Route
	routers := strings.Split(routeStr, ".")
	if len(routers) != 3 {
		// 如果路由格式不正确，返回错误
		return errors.New("route format unsupported")
	}

	// 获取服务器类型和处理方法
	serverType := routers[0]
	handlerMethod := fmt.Sprintf("%s.%s", routers[1], routers[2])
	logs.Info("receiver handler method:%v", handlerMethod)

	// 根据服务器类型获取对应的Connector配置
	connectorConfig := game.Conf.GetConnectorByServerType(serverType)
	if connectorConfig != nil {
		// 如果是本地connector服务器处理
		handler, ok := m.ConnectorHandlers[handlerMethod]
		if ok {
			// 调用处理函数并返回数据
			data, err := handler(c.GetSession(), message.Data)
			if err != nil {
				return err
			}
			// 将处理结果封装成响应消息
			marshal, _ := json.Marshal(data)
			message.Type = protocol.Response
			message.Data = marshal
			encode, err := protocol.MessageEncode(message)
			if err != nil {
				return err
			}
			// 编码并发送响应消息
			response, err := protocol.Encode(packet.Type, encode)
			if err != nil {
				return err
			}
			return c.SendMessage(response)
		}
	} else {
		// 如果不是本地connector服务器处理，则通过 NATS 进行远端调用
		dst, err := m.selectDst(serverType)
		if err != nil {
			logs.Error("remote send msg selectDst failed: ", err)
			return err
		}
		logs.Info("begin send message by nats, dst:%v, msg:%v", dst, message)
		// 构造远端调用消息
		msg := remote.Msg{
			Cid:         c.GetSession().Cid,
			Uid:         c.GetSession().Uid,
			Src:         m.ServerId,
			Dst:         dst,
			Router:      handlerMethod,
			Body:        message,
			SessionData: c.GetSession().data,
		}
		// 序列化消息并发送
		data, _ := json.Marshal(msg)
		err = m.RemoteClient.SendMsg(dst, data)
		if err != nil {
			logs.Error("remote send msg failed: ", err)
			return err
		}
	}
	return nil
}

// KickHandler 处理踢出消息
func (m *Manager) KickHandler(packet *protocol.Packet, c Connection) error {
	logs.Info("receiver KickHandler handler")
	return nil
}

// remoteReadChanHandler 用于接收通过nats订阅的消息
func (m *Manager) remoteReadChanHandler() {
	for {
		select {
		case body, ok := <-m.RemoteReadChan:
			if ok {
				logs.Info("sub nats read chan:%v", string(body))
				var msg remote.Msg
				if err := json.Unmarshal(body, &msg); err != nil {
					logs.Error("Unmarshal msg failed: ", err)
					continue
				}
				if msg.Type == remote.SessionType {
					// 需要特殊处理，Session类型是存储再connection中的session，并不推送
					m.setSessionData(msg)
					continue
				}
				if msg.Body != nil {
					if msg.Body.Type == protocol.Response || msg.Body.Type == protocol.Request {
						// 给客户端回消息，都是response
						msg.Body.Type = protocol.Response
						m.Response(&msg)
					}
					if msg.Body.Type == protocol.Push {
						m.RemotePushChan <- &msg
					}
				}
			}
		}
	}
}

// 选择目的地
func (m *Manager) selectDst(serverType string) (string, error) {
	serverConfigs, ok := game.Conf.ServersConf.TypeServer[serverType]
	if !ok {
		return "", errors.New("not found server")
	}
	// 随机一个目标服务
	rand.New(rand.NewSource(time.Now().UnixNano()))
	index := rand.Intn(len(serverConfigs))
	return serverConfigs[index].ID, nil

}

func (m *Manager) Response(msg *remote.Msg) {
	connection, ok := m.clients[msg.Cid]
	if !ok {
		logs.Info("%s client down，uid=%s", msg.Cid, msg.Uid)
		return
	}
	buf, err := protocol.MessageEncode(msg.Body)
	if err != nil {
		logs.Error("Response MessageEncode err:%v", err)
		return
	}
	res, err := protocol.Encode(protocol.Data, buf)
	if err != nil {
		logs.Error("Response Encode err:%v", err)
		return
	}
	if msg.Body.Type == protocol.Push {
		for _, v := range m.clients {
			if utils.Contains(msg.PushUser, v.GetSession().Uid) {
				logs.Info("Response Push User:%v", msg)
				v.SendMessage(res)
			}
		}
	} else {
		logs.Info("Response Push User:%v", msg)
		connection.SendMessage(res)
	}

}

func (m *Manager) RemotePushChanHandler() {
	for {
		select {
		case body, ok := <-m.RemotePushChan:
			if ok {
				logs.Info("nats push channel handler, body:%v", body)
				if body.Body.Type == protocol.Push {
					logs.Info("push ,body:%v", body)
					m.Response(body)
				}
			}

		}
	}
}

func (m *Manager) setSessionData(msg remote.Msg) {
	m.RLock()
	defer m.RUnlock()
	connection, ok := m.clients[msg.Cid]
	if ok {
		connection.GetSession().SetData(msg.Uid, msg.SessionData)
	}
}

// NewManager 创建一个新的 Manager 实例
func NewManager() *Manager {
	return &Manager{
		ClientReadChan: make(chan *MsgPack, 1024),
		clients:        make(map[string]Connection),
		handlers:       make(map[protocol.PackageType]EventHandler),
		RemoteReadChan: make(chan []byte, 1024),
		RemotePushChan: make(chan *remote.Msg, 1024),
	}
}
