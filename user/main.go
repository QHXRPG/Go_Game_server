package main

import (
	"common/config"
	"flag"
	"fmt"
)

// go run main.go -config=custom_config.yml
var configFile = flag.String("config", "application.yml", "config file")

func main() {
	// 读取配置文件
	flag.Parse()                   // 解析
	config.InitConfig(*configFile) // 加载配置
	fmt.Println(config.Conf)

	// 启动内存监控

	// 启动grpc服务端
}
