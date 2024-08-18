package app

import (
	"common/config"
	"common/logs"
	"context"
	"core/repo"
	"framework/node"
	"game/route"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Run 启动程序：启动 nats 服务
func Run(ctx context.Context, serverId string) error {
	// 初始化日志库
	logs.InitLog(config.Conf.AppName)

	// 定义一个退出函数
	exit := func() {}
	go func() {
		// 获取默认的连接器实例
		n := node.Default()
		exit = n.Close
		manager := repo.New()
		// 注册路由处理器给n
		n.RegisterHandler(route.Register(manager))
		// 启动连接器
		err := n.Run(serverId)
		if err != nil {
			logs.Error("server run err:%v", err)
			return
		}
	}()

	// 优雅启动与停止
	stop := func() {
		exit()
		// 给出时间让程序停止
		time.Sleep(3 * time.Second)
		logs.Info("stop app finish")
	}

	// 创建一个缓冲大小为 1 的通道，用于接收操作系统的信号
	c := make(chan os.Signal, 1)
	// 监听信号
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for {
		select {
		// 如果上下文被取消，执行停止操作
		case <-ctx.Done():
			stop()
			return nil
		// 接收到信号时执行相应的处理
		case s := <-c:
			switch s {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				stop()
				logs.Info("connector app exit")
				return nil
			case syscall.SIGHUP:
				stop()
				logs.Info("connector app (hang up)")
				return nil
			default:
				return nil
			}
		}
	}
}
