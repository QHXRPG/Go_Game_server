package discovery

//resolver 是一个包，用于 ⭐实现服务发现和负载均衡功能。
//它定义了 ⭐如何从服务注册中心etcd解析服务地址，
//并将这些地址 ⭐提供给 gRPC 客户端，以便客户端可以连接到这些服务。

import (
	"common/config"
	"common/logs"
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
	"time"
)

const schema = "etcd"

// Resolver 是一个etcd解析器，用于服务发现
type Resolver struct {
	// schema 是解析器的方案名称"etcd"，通常用于标识解析器的类型
	schema string

	// etcdCli 是etcd客户端，用于与etcd集群进行交互
	etcdCli *clientv3.Client

	// closeCh 是一个通道，用于在关闭解析器时通知相关的goroutine
	closeCh chan struct{}

	// DialTimeout 是与etcd服务器建立连接的超时时间，以秒为单位
	DialTimeout int

	// conf 是etcd的配置信息，包含etcd服务器地址、超时时间等
	conf config.EtcdConf

	// srvAddrList 是当前已解析的服务地址列表
	srvAddrList []resolver.Address

	// cc 是gRPC客户端连接，用于更新gRPC的连接状态
	cc resolver.ClientConn

	// key 是etcd中服务注册的前缀，用于定位服务地址
	key string

	// watchCh 是etcd的监听通道，用于监听服务地址的变化
	watchCh clientv3.WatchChan
}

// Build 用于创建etcd解析器，当grpc.Dial调用时，会触发此方法
func (r *Resolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r.cc = cc

	// 1. 创建etcd客户端
	var err error
	r.etcdCli, err = clientv3.New(clientv3.Config{
		Endpoints:   r.conf.Addrs,
		DialTimeout: time.Duration(r.DialTimeout) * time.Second,
	})
	if err != nil {
		logs.Fatal("connect etcd failed,err : %v", err)
	}
	r.closeCh = make(chan struct{})

	// 2. 根据key获取所有服务器地址
	r.key = target.URL.Host
	if err = r.sync(); err != nil {
		return nil, err
	}

	// 3. 监听节点有变动时更新
	go r.watch()
	return nil, nil
}

// Scheme 返回解析器的方案"etcd"
func (r *Resolver) Scheme() string {
	return r.schema
}

// Close 关闭解析器，停止监听
func (r *Resolver) Close() {
	r.closeCh <- struct{}{}
}

// watch 监听etcd中的节点变化
func (r *Resolver) watch() {
	// 定时器，每分钟同步一次数据
	ticker := time.NewTicker(time.Minute)
	// 监听节点的事件，从而触发不同的操作
	r.watchCh = r.etcdCli.Watch(context.Background(), r.key, clientv3.WithPrefix())
	for {
		select {
		case <-r.closeCh:
			// 关闭解析器
			logs.Info("close resolver, name=%s", r.key)
			return
		case res, ok := <-r.watchCh:
			// 处理etcd事件
			if ok {
				r.update(res.Events)
			}
		case <-ticker.C:
			// 定时同步数据
			if err := r.sync(); err != nil {
				logs.Error("resolver sync failed,err:%v", err)
			}
		}
	}
}

// sync 从etcd中获取当前所有服务地址并更新地址列表
func (r *Resolver) sync() error {
	// 创建一个带超时的上下文，超时时间为配置中的读写超时时间
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.conf.RWTimeout)*time.Second)
	defer cancel()

	// 从etcd中获取以r.key为前缀的所有键值对
	res, err := r.etcdCli.Get(ctx, r.key, clientv3.WithPrefix())
	if err != nil {
		// 如果获取失败，记录错误日志并返回错误
		logs.Error("get etcd register service failed, name=%s, err : %v", r.key, err)
		return err
	}
	r.srvAddrList = []resolver.Address{} // 清空当前的服务地址列表

	// 遍历获取到的键值对
	for _, v := range res.Kvs {
		// 解析键值对中的服务信息
		server, err := ParseValue(v.Value)
		if err != nil {
			// 如果解析失败，记录错误日志并跳过此键值对
			logs.Error("parse etcd register service failed, name=%s, err:%v", r.key, err)
			continue
		}
		// 将解析出的服务地址和权重添加到服务地址列表
		r.srvAddrList = append(r.srvAddrList, resolver.Address{
			Addr:       server.Addr,
			Attributes: attributes.New("weight", server.Weight),
		})
	}

	if len(r.srvAddrList) == 0 {
		logs.Error("get etcd register service failed, name=%s", r.key)
		return err
	}

	// 更新gRPC的客户端连接状态，将新的服务地址列表传入
	err = r.cc.UpdateState(resolver.State{Addresses: r.srvAddrList})
	if err != nil {
		// 如果更新状态失败，记录错误日志并返回错误
		logs.Error("updateState etcd register service failed, name=%s, err : %v", r.key, err)
		return err
	}

	// 返回nil表示同步成功
	return nil
}

// update 根据etcd事件更新服务地址列表
func (r *Resolver) update(events []*clientv3.Event) {
	// ev :从 etcd 接收到的事件
	for _, ev := range events {
		var server Server
		var err error
		switch ev.Type {
		case clientv3.EventTypePut:
			// 处理新增或更新事件
			server, err = ParseValue(ev.Kv.Value)
			if err != nil {
				logs.Error("[clientv3.EventTypePut]parse etcd register service failed, name=%s, err:%v", r.key, err)
				continue
			}
			// 构建服务地址结构体
			addr := resolver.Address{
				Addr:       server.Addr,
				Attributes: attributes.New("weight", server.Weight),
			}
			// 如果地址列表中不存在该地址，则添加
			if !Exist(r.srvAddrList, addr) {
				r.srvAddrList = append(r.srvAddrList, addr)
				// 更新gRPC客户端连接的状态
				err = r.cc.UpdateState(resolver.State{Addresses: r.srvAddrList})
				if err != nil {
					logs.Error("[clientv3.EventTypePut]updateState etcd register service failed, name=%s, err : %v", r.key, err)
				}
			}
		case clientv3.EventTypeDelete:
			// 处理删除事件
			server, err := ParseKey(string(ev.Kv.Key))
			if err != nil {
				logs.Error("[clientv3.EventTypeDelete] parse key, name=%s, err:%v", r.key, err)
				continue
			}
			// 构建服务地址结构体
			addr := resolver.Address{Addr: server.Addr}
			// 从地址列表中删除该地址
			if list, ok := Remove(r.srvAddrList, addr); ok {
				r.srvAddrList = list
				// 更新gRPC客户端连接的状态
				err := r.cc.UpdateState(resolver.State{Addresses: r.srvAddrList})
				if err != nil {
					logs.Error("[clientv3.EventTypeDelete] updateState etcd register service failed, name=%s, err : %v", r.key, err)
				}
			}
		}
	}
}

// Remove 从地址列表中移除指定地址
func Remove(list []resolver.Address, addr resolver.Address) ([]resolver.Address, bool) {
	for i := range list {
		if list[i].Addr == addr.Addr {
			list[i] = list[len(list)-1] // 将最后一个元素移到当前位置
			return list[:len(list)-1], true
		}
	}
	return nil, false
}

// Exist 检查地址列表中是否存在指定地址
func Exist(list []resolver.Address, addr resolver.Address) bool {
	for i := range list {
		if list[i].Addr == addr.Addr {
			return true
		}
	}
	return false
}

// NewResolver 创建一个新的etcd解析器实例
func NewResolver(conf config.EtcdConf) *Resolver {
	return &Resolver{
		schema:      schema,
		DialTimeout: conf.DialTimeout,
		conf:        conf,
	}
}
