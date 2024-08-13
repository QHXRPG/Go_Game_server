package discovery

// ⭐ 服务注册器，将 gRPC 服务注册到 etcd，并通过租约和心跳机制保持服务的可用性。

import (
	"common/config"
	"common/logs"
	"context"
	"encoding/json"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

// Register 用于将 gRPC 服务注册到 etcd
// 原理：创建一个租约，gRPC 服务注册到 etcd，绑定租约。
// 过了租约时间，etcd 就会删除 gRPC 服务信息。
// 通过心跳机制实现续租，如果租约失效则重新注册。

// Register 用于管理服务注册信息和与 etcd 进行交互的相关参数，如租约、连接超时、心跳响应等
type Register struct {
	etcdCli     *clientv3.Client                        // etcd 客户端
	leaseId     clientv3.LeaseID                        // 租约 ID
	Dialtimeout int                                     // 连接超时时间
	ttl         int                                     // 租约时间（秒）
	keepAliveCh <-chan *clientv3.LeaseKeepAliveResponse // 心跳响应通道
	info        Server                                  // 注册的 Server 信息
	closeCh     chan struct{}                           // 用于关闭注册器的通道
}

// NewRegister 创建一个新的注册器实例
func NewRegister() *Register {
	return &Register{
		Dialtimeout: 3,
	}
}

// Register 将服务注册到 etcd
func (r *Register) Register(conf config.EtcdConf) error {
	// 构建注册信息
	info := Server{
		Name:    conf.Register.Name,
		Addr:    conf.Register.Addr,
		Weight:  conf.Register.Weight,
		Version: conf.Register.Version,
		Ttl:     conf.Register.Ttl,
	}

	// 建立 etcd 连接
	var err error
	r.etcdCli, err = clientv3.New(clientv3.Config{
		Endpoints:   conf.Addrs, // etcd 集群的地址
		DialTimeout: time.Duration(r.Dialtimeout) * time.Second,
	})
	if err != nil {
		return err
	}
	r.info = info

	// 注册服务
	err = r.register()
	if err != nil {
		return err
	}

	// 初始化关闭通道
	r.closeCh = make(chan struct{})

	// 启动协程，监控租约心跳
	go r.watcher()
	return nil
}

// Close 关闭注册器，注销服务
func (r *Register) Close() {
	r.closeCh <- struct{}{}
}

// register 注册服务到 etcd，并绑定租约
func (r *Register) register() error {
	// 创建租约
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Dialtimeout)*time.Second)
	defer cancel()

	// 创建租约
	err := r.createLease(ctx, r.info.Ttl)
	if err != nil {
		return err
	}

	// 启动心跳检测
	if r.keepAliveCh, err = r.keepAlive(); err != nil {
		return err
	}

	// 绑定租约，将服务信息注册到 etcd
	data, _ := json.Marshal(r.info)
	return r.bindLease(ctx, r.info.BuildRegisterKey(), string(data))
}

// bindLease 将服务信息绑定到租约
func (r *Register) bindLease(ctx context.Context, key string, value string) error {
	_, err := r.etcdCli.Put(ctx, key, value, clientv3.WithLease(r.leaseId))
	if err != nil {
		logs.Error("bind register err:", err)
		return err
	}
	return nil
}

// createLease 创建一个指定 ttl 的租约
func (r *Register) createLease(ctx context.Context, ttl int64) error {
	grant, err := r.etcdCli.Grant(ctx, ttl)
	if err != nil {
		logs.Error("create grant err:", err)
		return err
	}
	r.leaseId = grant.ID
	return nil
}

// keepAlive 启动租约的心跳检测
func (r *Register) keepAlive() (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	// 如果设置了超时，长连接就会断掉，不要设置超时
	// 不停地发消息，保持租约（续租）
	keepAliveResponses, err := r.etcdCli.KeepAlive(context.Background(), r.leaseId)
	if err != nil {
		logs.Error("keep alive err:", err)
		return keepAliveResponses, err
	}
	return keepAliveResponses, nil
}

// watcher 监控租约心跳，处理续约和重新注册逻辑
func (r *Register) watcher() {
	// 定时器，用于定期检查租约状态
	ticker := time.NewTicker(time.Duration(r.info.Ttl) * time.Second)
	for {
		select {
		case <-r.closeCh:
			// 收到关闭信号，注销服务并撤销租约
			err := r.unregister()
			if err != nil {
				logs.Error("unregister err:", err)
			}
			// 撤销租约
			_, err = r.etcdCli.Revoke(context.Background(), r.leaseId)
			if err != nil {
				logs.Error("revoke err:", err)
			}
			if r.etcdCli != nil {
				err := r.etcdCli.Close()
				if err != nil {
					logs.Error("close etcd err:", err)
					return
				}
			}
			logs.Info("close etcd client")
			return
		case <-r.keepAliveCh:
			// 收到心跳响应，如果响应为空，重新注册服务
		case <-ticker.C:
			// 定时检查心跳通道是否为空，如果为空，重新注册服务
			if r.keepAliveCh == nil {
				if err := r.register(); err != nil {
					logs.Error("ticker register err:", err)
				}
			}
		}
	}
}

// unregister 注销服务
func (r *Register) unregister() error {
	_, err := r.etcdCli.Delete(context.Background(), r.info.BuildRegisterKey())
	if err != nil {
		logs.Error("unregister err:", err)
	}
	return err
}
