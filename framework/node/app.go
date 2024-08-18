package node

import (
	"common/logs"
	"encoding/json"
	"framework/remote"
)

// App 就是nats的客户端，处理实际游戏逻辑的服务
type App struct {
	remoteClient remote.Client
	readChan     chan []byte
	writeChan    chan *remote.Msg
	handlers     LogicHandler
}

func Default() *App {
	return &App{
		readChan:  make(chan []byte),
		writeChan: make(chan *remote.Msg, 1024),
		handlers:  make(LogicHandler),
	}
}

func (a *App) Run(serverId string) error {
	a.remoteClient = remote.NewNatsClient(serverId, a.readChan)
	err := a.remoteClient.Run()
	if err != nil {
		logs.Error("remoteClient run err:", err)
		return err
	}
	go a.readChanMsg()
	go a.writeChanMsg()
	return nil
}

func (a *App) readChanMsg() {
	// 收到的是其它 nats client发送的消息
	logs.Info("readChanMsg start")
	for {
		select {
		case msg := <-a.readChan:
			var remoteMsg remote.Msg
			json.Unmarshal(msg, &remoteMsg)
			session := remote.NewSession(a.remoteClient, &remoteMsg)
			session.SetData(remoteMsg.SessionData)

			// 根据路由消息， 发送给对应的handler处理
			router := remoteMsg.Router
			if handlerFunc := a.handlers[router]; handlerFunc != nil {
				result := handlerFunc(session, remoteMsg.Body.Data)
				message := remoteMsg.Body
				var body []byte
				if result != nil {
					body, _ = json.Marshal(result)
				}
				message.Data = body

				// 得到结果，发送给connector
				responseMsg := &remote.Msg{
					Src:  remoteMsg.Dst,
					Dst:  remoteMsg.Src,
					Body: message,
					Uid:  remoteMsg.Uid,
					Cid:  remoteMsg.Cid,
				}
				a.writeChan <- responseMsg
			}
		}
	}
}

func (a *App) writeChanMsg() {
	for {
		select {
		case msg, ok := <-a.writeChan:
			if ok {
				marshal, _ := json.Marshal(msg)
				err := a.remoteClient.SendMsg(msg.Dst, marshal)
				if err != nil {
					logs.Error("app remotr send msg err", err)
				}
			}

		}
	}
}

func (a *App) Close() {
	if a.remoteClient != nil {
		a.remoteClient.Close()
	}
}

func (a *App) RegisterHandler(handler LogicHandler) {
	a.handlers = handler
}
