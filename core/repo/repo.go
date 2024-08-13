package repo

import "common/database"

// Manager 结构体包含了 MongoDB 和 Redis 的管理器实例
type Manager struct {
	Mongo *database.MongoManger
	Redis *database.RedisManger
}

// Close 方法用于关闭 MongoDB 和 Redis 的连接
func (m Manager) Close() {
	// 检查 Mongo 是否非空，如果是则调用其 Close 方法关闭连接
	if m.Mongo != nil {
		m.Mongo.Close()
	}
	// 检查 Redis 是否非空，如果是则调用其 Close 方法关闭连接
	if m.Redis != nil {
		m.Redis.Close()
	}
}

// New 函数用于创建并返回一个新的 Manager 实例
func New() *Manager {
	return &Manager{
		// 初始化 MongoDB 管理器
		Mongo: database.NewMongo(),
		// 初始化 Redis 管理器
		Redis: database.NewRedis(),
	}
}
