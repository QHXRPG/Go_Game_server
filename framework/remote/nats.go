package remote

import (
	"common/logs"
	"github.com/nats-io/nats.go"
)

type NatsClient struct {
	serverId string
	conn     *nats.Conn
	readChan chan []byte
}

func (c *NatsClient) SendMsg(dst string, data []byte) error {
	if c.conn != nil {
		return c.conn.Publish(dst, data) // 向目标服务推送消息
	}
	return nil
}

func NewNatsClient(serverId string, readChan chan []byte) *NatsClient {
	return &NatsClient{
		serverId: serverId,
		readChan: readChan,
	}
}

func (c *NatsClient) Run() error {
	var err error
	c.conn, err = nats.Connect("nats://0.0.0.0:4222")
	if err != nil {
		logs.Error("Nats connect err:", err)
		return err
	}
	go c.sub() // 订阅
	return nil
}

func (c *NatsClient) Close() error {
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

func (c *NatsClient) sub() {
	_, err := c.conn.Subscribe(c.serverId, func(msg *nats.Msg) {
		// 收到其它nats client发送的消息
		logs.Info("serverId:%v sub msg:%v", c.serverId, string(msg.Data))
		c.readChan <- msg.Data
	})
	if err != nil {
		logs.Error("nats Subscribe err:", err)
	}
}
