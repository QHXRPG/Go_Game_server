package main

import (
	"common/config"
	"common/metrics"
	"connector/app"
	"context"
	"fmt"
	"framework/game"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "connector",
	Short: "connector 管理连接，session以及路由请求",
	Long:  `connector 管理连接，session以及路由请求`,
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

	// 启动连接器
	err := app.Run(context.Background(), serverId)
	if err != nil {
		log.Println(err)
		os.Exit(-1)
	}
}
