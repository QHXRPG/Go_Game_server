package main

import (
	"common/config"
	"common/metrics"
	"context"
	"flag"
	"fmt"
	"gate/app"
	"log"
	"os"
)

// go run main.go -config=custom_config.yml
var configFile = flag.String("config", "application.yml", "config file")

func main() {
	// 读取配置文件
	flag.Parse()                   // 解析
	config.InitConfig(*configFile) // 加载配置

	// 启动内存监控, 放入协程当中启动
	// 点击：http://localhost:5854/debug/statsviz
	go func() {
		err := metrics.Serve(fmt.Sprintf("0.0.0.0:%d", config.Conf.MetricPort))
		if err != nil {
			panic(err)
		}
	}()

	// 启动grpc服务端
	err := app.Run(context.Background())
	if err != nil {
		log.Println(err)
		os.Exit(-1)
	}
}
