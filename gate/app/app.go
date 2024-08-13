package app

import (
	"common/config"
	"common/logs"
	"context"
	"fmt"
	"gate/router"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Run 启动程序：启动grpc服务 /启动http服务 /启动日志 /启动数据库
func Run(ctx context.Context) error {
	logs.InitLog(config.Conf.AppName) // 做一个日志库
	go func() {
		// gin 启动  注册一个路由
		r := router.RegisterRouter()

		// http接口
		err := r.Run(fmt.Sprintf(":%d", config.Conf.HttpPort))
		if err != nil {
			logs.Fatal("gate gin run err:", err)
		}
	}()

	// 优雅启动与停止: 信号
	stop := func() {
		time.Sleep(3 * time.Second) // 给出时间让程序停止
		logs.Info("stop app finish")
	}
	// 监听信号
	c := make(chan os.Signal, 1) // 创建了一个缓冲大小为 1 的通道, 用于接收操作系统的信号
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
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
