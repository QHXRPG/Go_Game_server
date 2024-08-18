package remote

import (
	"common/logs"
	"encoding/json"
	"framework/protocol"
	"sync"
)

// Session 存储当前玩家的相关信息
type Session struct {
	sync.RWMutex
	client          Client // 当前客户端
	msg             *Msg   // 消息
	pushChan        chan *userPushMsg
	data            map[string]any
	pushSessionChan chan map[string]any
}

// 推送的消息内容是什么，路由是什么
type pushMsg struct {
	data   []byte
	router string
}

// 给哪些用户推送什么消息
type userPushMsg struct {
	PushMsg pushMsg  `json:"pushMsg"`
	Users   []string `json:"users"`
}

func NewSession(client Client, msg *Msg) *Session {
	s := &Session{
		client:          client,
		msg:             msg,
		pushChan:        make(chan *userPushMsg, 1024),
		data:            make(map[string]any),
		pushSessionChan: make(chan map[string]any, 1024),
	}
	// 开一个协程，专门去读channel中的消息
	go s.pushChanRead()
	go s.pushSessionChanRead()
	return s
}

func (s *Session) GetUid() string {
	return s.msg.Uid
}

// Push 将房间ID发给玩家
func (s *Session) Push(user []string, pushMsgData any, route string) {
	msg, _ := json.Marshal(pushMsgData)
	_pushMsg := pushMsg{
		data:   msg,
		router: route,
	}
	_userPushMsg := &userPushMsg{
		Users:   user,
		PushMsg: _pushMsg,
	}
	// 将包含房间ID的消息送进pushChan通道中
	logs.Info("push msg:%v", _userPushMsg)
	s.pushChan <- _userPushMsg
}

// 从通道读取数据并发送消息，处理推送逻辑。
func (s *Session) pushChanRead() {
	for {
		select {
		case data := <-s.pushChan:
			pushMessage := protocol.Message{
				Type:  protocol.Push,
				ID:    s.msg.Body.ID,
				Route: data.PushMsg.router,
				Data:  data.PushMsg.data,
			}
			msg := Msg{
				Dst:      s.msg.Src,
				Src:      s.msg.Dst,
				Body:     &pushMessage,
				Cid:      s.msg.Cid,
				PushUser: data.Users,
			}
			result, _ := json.Marshal(msg)
			logs.Info("push msg dst:%v", msg.Dst)
			err := s.client.SendMsg(msg.Dst, result)
			if err != nil {
				logs.Info("push msg err:%v, msg=%v", err, msg)
			}
		}
	}
}

func (s *Session) Put(key string, value any) {
	s.Lock()
	defer s.Unlock()
	s.data[key] = value
	s.pushSessionChan <- s.data // 将新添加的数据同步到Nats
}

func (s *Session) pushSessionChanRead() {
	for {
		select {
		case data := <-s.pushSessionChan:
			_pushMsg := Msg{
				Dst:         s.msg.Src,
				Src:         s.msg.Dst,
				Cid:         s.msg.Cid,
				Uid:         s.msg.Uid,
				SessionData: data,
				Type:        SessionType,
			}
			res, _ := json.Marshal(_pushMsg)
			err := s.client.SendMsg(_pushMsg.Dst, res)
			if err != nil {
				logs.Error("push msg err:%v, msg=%v", err, _pushMsg)
				return
			}
		}
	}
}

func (s *Session) SetData(data map[string]any) {
	s.Lock()
	defer s.Unlock()
	for k, v := range data {
		s.data[k] = v
	}
}

func (s *Session) Get(key string) (any, bool) {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.data[key]
	return v, ok
}
