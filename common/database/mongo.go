package database

import (
	"common/config"
	"common/logs"
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

type MongoManger struct {
	Client *mongo.Client
	Db     *mongo.Database
}

func NewMongo() *MongoManger {
	// 创建一个带有超时机制的上下文, 限制某个操作的执行时间，例如网络请求、数据库查询等。
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel() //当前函数返回（或中断）时自动调用以释放资源并避免上下文泄漏。

	// 加载配置文件所提供的Mongodb的URL
	clientOptions := options.Client().ApplyURI(config.Conf.Database.MongoConf.Url)
	clientOptions.SetMinPoolSize(uint64(config.Conf.Database.MongoConf.MinPoolSize))
	clientOptions.SetMinPoolSize(uint64(config.Conf.Database.MongoConf.MinPoolSize))

	// 连接到 MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logs.Fatal("mongo connect err:", err)
		return nil
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		logs.Fatal("mongo connect err:", err)
	}

	m := &MongoManger{
		Client: client,
	}
	m.Db = m.Client.Database(config.Conf.Database.MongoConf.Db) // 拿到配置文件中Mongo数据库的名字
	return m
}

func (m *MongoManger) Close() {
	err := m.Client.Disconnect(context.TODO())
	if err != nil {
		logs.Error("mongo disconnect err:", err)
	}
}
