package app

import (
	"common/config"
	"common/discovery"
	"common/logs"
	"context"
	"core/repo"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	"user/internal/service"
	"user/pb"
)

// Run 启动程序：启动grpc服务 /启动http服务 /启动日志 /启动数据库
// ctx 的作用主要是用来接收取消信号
// 通过 ctx.Done() 通道可以知道什么时候该停止执行当前的操作
func Run(ctx context.Context) error {
	logs.InitLog(config.Conf.AppName)   // 做一个日志库
	register := discovery.NewRegister() // etcd注册中心 将grpc服务注册到etcd中， 客户端访问时候通过etcd获取grpc的地址
	server := grpc.NewServer()          // 启动grpc服务端
	manager := repo.New()               // 初始化数据库 mongo redis
	go func() {
		lis, err := net.Listen("tcp", config.Conf.Grpc.Addr) // 阻塞操作，需要放到一个协程当中
		if err != nil {
			logs.Fatal("failed to listen:", err)
		}

		// 注册 grpc service 到 etcd （mongo、redis）
		err = register.Register(config.Conf.Etcd)
		if err != nil {
			logs.Fatal("failed to register etcd:", err)
		}

		// 将创建新的账号服务 注册到 gRPC 服务器
		pb.RegisterUserServiceServer(server, service.NewAccountService(manager))
		err = server.Serve(lis) // 启动 gRPC 服务器并监听。(阻塞)
		if err != nil {
			logs.Fatal("failed to serve:", err)
		}
	}()

	// 优雅启动与停止: 信号
	stop := func() {
		server.Stop()
		register.Close()
		manager.Close()
		time.Sleep(3 * time.Second) // 给出时间让程序停止
		logs.Info("stop app finish")
	}
	// 监听信号
	// make用于分配和初始化一些类型的数据结构, chan 是 Go 语言中的通道类型
	c := make(chan os.Signal, 1) // 创建了一个缓冲大小为 1 的通道, 用于接收操作系统的信号
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// <- 是通道操作符，用于发送和接收通道中的数据
	for {
		select {
		case <-ctx.Done():
			stop()
			return nil
		case s := <-c:
			switch s {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				stop()
				logs.Info("user app exit")
				return nil
			case syscall.SIGHUP:
				stop()
				logs.Info("user app (hang up)")
				return nil
			default:
				return nil
			}
		}
	}
}
