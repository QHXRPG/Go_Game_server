package rpc

import (
	"common/config"
	"common/discovery"
	"common/logs"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"user/pb"
)

// UserClient 是一个全局的用户服务客户端实例
var (
	UserClient pb.UserServiceClient
)

// Init 初始化 gRPC 客户端连接
func Init() {
	r := discovery.NewResolver(config.Conf.Etcd) // 创建一个新的etcd解析器实例
	resolver.Register(r)                         // 注册解析器
	domain := config.Conf.Domain["user"]         // 获取用户服务的域名配置

	// 初始化用户服务客户端
	initClient(r.Scheme(), domain.Name, domain.LoadBalance, &UserClient)
}

// initClient 初始化 gRPC 客户端连接
// scheme 是解析器方案"etcd"，name 是服务名称，loadBalance 表示是否启用负载均衡，client 是客户端实例
func initClient(scheme, name string, loadBalance bool, client interface{}) {
	// 构建服务地址
	addr := fmt.Sprintf("%s:///%s", scheme, name)

	// 创建 gRPC 连接选项
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()), // 使用不安全的传输凭证（不加密）
	}

	// 如果启用负载均衡，则设置负载均衡策略为轮询
	if loadBalance {
		opts = append(opts, grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, "round_robin")))
	}

	// 创建 gRPC 连接
	conn, err := grpc.DialContext(context.TODO(), addr, opts...)
	if err != nil {
		logs.Fatal("rpc connect etcd error: %v", err)
	}

	// 根据客户端类型初始化具体的客户端实例
	switch c := client.(type) {
	case *pb.UserServiceClient:
		*c = pb.NewUserServiceClient(conn)
	default:
		logs.Fatal("unsupported client type")
	}
}
