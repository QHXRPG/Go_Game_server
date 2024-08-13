# Go_Game_server
基于go语言实现的游戏服务器

1. 使用statsviz第三方库进行网络监测。
2. 使用信号控制grpc服务的启动与停止，并通过etcd实现负载均衡。
3. 在网关构建grpc客户端，通过tcp请求USER的登录注册等服务。
4. 通过docker启动nats服务，使得connector服务能够与hall服务互相订阅与转发来自客户端的消息，实现松耦合。
