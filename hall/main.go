package main

import (
	"common/config"
	"common/metrics"
	"context"
	"fmt"
	"framework/game"
	"github.com/spf13/cobra"
	"hall/app"
	"log"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "hall",
	Short: "hall 大厅相关的处理",
	Long:  `hall 大厅相关的处理`,
	Run: func(cmd *cobra.Command, args []string) {
	},
	PostRun: func(cmd *cobra.Command, args []string) {
	},
}

// go run main.go -config=custom_config.yml
//var configFile = flag.String("config", "application.yml", "config file")

var (
	configFile    string
	gameConfigDir string
	serverId      string
)

func init() {
	rootCmd.Flags().StringVar(&configFile, "config", "application.yml", "app config yml file")
	rootCmd.Flags().StringVar(&gameConfigDir, "gameDir", "../config", "game config dir")
	rootCmd.Flags().StringVar(&serverId, "serverId", "", "app server id， required")
	_ = rootCmd.MarkFlagRequired("serverId")
}

// 连接 写一个websocket的连接， 客户端需要连接这个websocket
// 1.websocket_manager 2.natsClient
func main() {
	// 读取配置文件
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// 加载配置
	config.InitConfig(configFile)
	game.InitConfig(gameConfigDir)

	// 启动内存监控, 放入协程当中启动
	// 点击：http://localhost:5854/debug/statsviz
	go func() {
		err := metrics.Serve(fmt.Sprintf("0.0.0.0:%d", config.Conf.MetricPort))
		if err != nil {
			panic(err)
		}
	}()

	// 连接nats服务
	err := app.Run(context.Background(), serverId)
	if err != nil {
		log.Println(err)
		os.Exit(-1)
	}
}
