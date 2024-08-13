package net

import "sync"

// Session 表示一个会话，包含会话ID、用户ID和数据
type Session struct {
	sync.RWMutex                // 嵌入读写锁，用于保护并发访问
	Cid          string         // 会话ID
	Uid          string         // 用户ID
	data         map[string]any // 存储会话数据的字典
}

// NewSession 创建一个新的 Session 实例
func NewSession(cid string) *Session {
	return &Session{
		Cid:  cid,
		data: make(map[string]any), // 初始化 data 字典
	}
}

// Put 向会话中添加一个键值对
func (s *Session) Put(key string, value any) {
	s.Lock()         // 加写锁，保护 data 字典的并发写操作
	defer s.Unlock() // 确保在函数退出时释放锁
	s.data[key] = value
}

// Get 从会话中获取一个键对应的值
func (s *Session) Get(key string) (any, bool) {
	s.RLock() // 加读锁，保护 data 字典的并发读操作
	defer s.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

// SetData 批量设置会话数据
func (s *Session) SetData(uid string, data map[string]any) {
	s.Lock()
	defer s.Unlock()
	if s.Uid == uid {
		for k, v := range data {
			s.data[k] = v
		}
	}
}
