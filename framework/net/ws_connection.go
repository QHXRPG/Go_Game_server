package net

import (
	"common/logs"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"sync/atomic"
	"time"
)

// 这个文件定义了 WsConnection 结构体，用于管理 WebSocket 连接的生命周期，
// 包括消息的读取和发送、心跳检测等。
// 提供了对 WebSocket 连接的封装，支持消息的异步读写和连接的管理。
// 通过 WsConnection，可以方便地处理客户端和服务器之间的实时通信。

var cidBase int64 = 10000

var (
	maxMessageSize int64 = 1024
	pongWait             = 10 * time.Second
	writeWait            = 10 * time.Second
	pingInterval         = (pongWait * 9) / 10
)

// WsConnection 结构体管理 WebSocket 连接
type WsConnection struct {
	Cid        string
	Conn       *websocket.Conn
	manager    *Manager
	ReadChan   chan *MsgPack
	WriteChan  chan []byte
	Session    *Session
	pingTicker *time.Ticker
}

// GetSession 获取会话
func (c WsConnection) GetSession() *Session {
	return c.Session
}

// SendMessage 发送消息到客户端
func (c WsConnection) SendMessage(buf []byte) error {
	c.WriteChan <- buf
	return nil
}

// Close 关闭连接
func (c WsConnection) Close() {
	if c.Conn != nil {
		err := c.Conn.Close()
		if err != nil {
			return
		}
	}
	if c.pingTicker != nil {
		c.pingTicker.Stop()
	}
}

// Run 启动读写消息和心跳检测
func (c WsConnection) Run() {
	go c.readMessage()
	go c.WriteMessage()

	// 心跳检测，WebSocket 中的 ping pong 机制
	c.Conn.SetPongHandler(c.PongHandler)
}

// WriteMessage 向客户端发送消息
func (c WsConnection) WriteMessage() {
	c.pingTicker = time.NewTicker(pingInterval)
	for {
		select {
		case message, ok := <-c.WriteChan:
			if !ok {
				if err := c.Conn.WriteMessage(websocket.CloseMessage, nil); err != nil {
					logs.Error("connection closed, %v", err)
				}
				return
			}
			if err := c.Conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				logs.Error("client[%s] write message err :%v", c.Cid, err)
			}
		case <-c.pingTicker.C:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logs.Error("client[%s] ping SetWriteDeadline err :%v", c.Cid, err)
			}
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logs.Error("client[%s] ping  err :%v", c.Cid, err)
				c.Close()
			}
		}
	}
}

// readMessage 接收客户端发来的消息
func (c WsConnection) readMessage() {
	defer func() {
		c.manager.removeClient(c)
	}()
	c.Conn.SetReadLimit(maxMessageSize)

	// 设置 WebSocket 连接的读取超时时间
	if err := c.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		logs.Error("client[%s] write ping SetReadDeadline error:%v", c.Cid, err)
	}
	for {
		messageType, message, err := c.Conn.ReadMessage() // 库函数
		if err != nil {
			break
		}
		// 客户端发来的消息是二进制消息
		if messageType == websocket.BinaryMessage {
			if c.ReadChan != nil {
				// 通过协程一直读 channel 的消息，将读到的消息发到 ReadChan 中去
				c.ReadChan <- &MsgPack{
					Cid:  c.Cid,
					Body: message,
				}
			}
		} else {
			logs.Fatal("unsupported message type:", messageType)
		}
	}
}

// PongHandler 处理 pong 消息，更新读取超时时间
func (c WsConnection) PongHandler(data string) error {
	if err := c.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return err
	}
	return c.Conn.WriteMessage(websocket.PongMessage, []byte(data))
}

// NewWsConnection 创建一个新的 WsConnection 实例
func NewWsConnection(conn *websocket.Conn, manager *Manager) *WsConnection {
	cid := fmt.Sprintf("%s-%s-%d", uuid.New().String(), manager.ServerId, atomic.AddInt64(&cidBase, 1))
	return &WsConnection{
		Conn:      conn,
		manager:   manager,
		Cid:       cid,
		WriteChan: make(chan []byte, 1024), // 有缓冲 channel
		ReadChan:  manager.ClientReadChan,
		Session:   NewSession(cid),
	}
}
