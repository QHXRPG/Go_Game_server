package game

import (
	"common/logs"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"io"
	"log"
	"os"
	"path"
)

// Conf 全局配置变量
var Conf *Config

// 配置文件名称常量
const (
	gameConfig = "gameConfig.json"
	servers    = "servers.json"
)

// Config 定义了游戏服务器的配置结构
type Config struct {
	GameConfig  map[string]GameConfigValue `json:"gameConfig"`
	ServersConf ServersConf                `json:"serversConf"`
}

// ServersConf 定义了服务器相关的配置结构
type ServersConf struct {
	Nats       NatsConfig         `json:"nats"`
	Connector  []*ConnectorConfig `json:"connector"`
	Servers    []*ServersConfig   `json:"servers"`
	TypeServer map[string][]*ServersConfig
}

// ServersConfig 定义了单个服务器的配置
type ServersConfig struct {
	ID               string `json:"id"`
	ServerType       string `json:"serverType"`
	HandleTimeOut    int    `json:"handleTimeOut"`
	RPCTimeOut       int    `json:"rpcTimeOut"`
	MaxRunRoutineNum int    `json:"maxRunRoutineNum"`
}

// ConnectorConfig 定义了Connector的配置
type ConnectorConfig struct {
	ID         string `json:"id"`
	Host       string `json:"host"`
	ClientPort int    `json:"clientPort"`
	Frontend   bool   `json:"frontend"`
	ServerType string `json:"serverType"`
}

// NatsConfig 定义了NATS服务器的配置
type NatsConfig struct {
	Url string `json:"url"`
}

// InitConfig 初始化配置，从指定的配置目录加载配置文件
func InitConfig(configDir string) {
	Conf = new(Config)
	// 从配置目录下加载gameConfig.json和servers.json配置文件
	dir, err := os.ReadDir(configDir)
	if err != nil {
		logs.Fatal("read config dir err: %v", err)
	}
	for _, v := range dir {
		configFile := path.Join(configDir, v.Name())

		// 读取 gameConfig.json
		if v.Name() == gameConfig {
			readGameConfig(configFile)
		}

		// 读取 servers.json
		if v.Name() == servers {
			readServersConfig(configFile)
		}
	}
}

// readServersConfig 读取并解析servers.json配置文件
func readServersConfig(configFile string) {
	var serversConf ServersConf
	v := viper.New()
	v.SetConfigFile(configFile)
	v.WatchConfig() // 监控配置文件变化
	v.OnConfigChange(func(e fsnotify.Event) {
		log.Println("serversConf配置文件被修改")
		err := v.Unmarshal(&serversConf)
		if err != nil {
			panic(fmt.Errorf("serversConf配置文件被修改以后，报错，err:%v \n", err))
		}
		Conf.ServersConf = serversConf
		typeServerConfig()
	})
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("读取serversConf配置文件报错，err:%v \n", err))
	}
	if err := v.Unmarshal(&serversConf); err != nil {
		panic(fmt.Errorf("Unmarshal data to Conf failed ，err:%v \n", err))
	}
	Conf.ServersConf = serversConf
	typeServerConfig()
}

// typeServerConfig 根据服务器类型分类配置
func typeServerConfig() {
	if len(Conf.ServersConf.Servers) > 0 {
		if Conf.ServersConf.TypeServer == nil {
			Conf.ServersConf.TypeServer = make(map[string][]*ServersConfig)
		}
		for _, v := range Conf.ServersConf.Servers {
			if Conf.ServersConf.TypeServer[v.ServerType] == nil {
				Conf.ServersConf.TypeServer[v.ServerType] = make([]*ServersConfig, 0, 10)
			}
			Conf.ServersConf.TypeServer[v.ServerType] = append(Conf.ServersConf.TypeServer[v.ServerType], v)
		}
	}
}

// GameConfigValue 定义了游戏配置的值类型
type GameConfigValue map[string]any

// readGameConfig 读取并解析gameConfig.json配置文件
func readGameConfig(configFile string) {
	//var gameConfig = make(map[string]GameConfigValue)
	//v := viper.New()
	//v.SetConfigFile(configFile)
	//v.WatchConfig() // 监控配置文件变化
	//v.OnConfigChange(func(e fsnotify.Event) {
	//	log.Println("gameConfig配置文件被修改")
	//	err := v.Unmarshal(&gameConfig)
	//	if err != nil {
	//		panic(fmt.Errorf("gameConfig配置文件被修改以后，报错，err:%v \n", err))
	//	}
	//	Conf.GameConfig = gameConfig
	//})
	//err := v.ReadInConfig()
	//if err != nil {
	//	panic(fmt.Errorf("读取gameConfig配置文件报错，err:%v \n", err))
	//}
	//if err := v.Unmarshal(&gameConfig); err != nil {
	//	panic(fmt.Errorf("Unmarshal data to Conf failed ，err:%v \n", err))
	//}
	//Conf.GameConfig = gameConfig
	file, err := os.Open(configFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	var gc map[string]GameConfigValue
	err = json.Unmarshal(data, &gc)
	if err != nil {
		panic(err)
	}
	Conf.GameConfig = gc
}

// GetConnector 根据服务器ID获取对应的Connector配置
func (c *Config) GetConnector(serverId string) *ConnectorConfig {
	for _, v := range c.ServersConf.Connector {
		if v.ID == serverId {
			return v
		}
	}
	return nil
}

// GetConnectorByServerType 根据服务器类型获取对应的Connector配置
func (c *Config) GetConnectorByServerType(serverType string) *ConnectorConfig {
	for _, v := range c.ServersConf.Connector {
		if v.ServerType == serverType {
			return v
		}
	}
	return nil
}

// GetFromGameConfig 根据服务器类型获取对应的Connector配置
func (c *Config) GetFromGameConfig() map[string]any {
	result := make(map[string]any)
	for k, v := range c.GameConfig {
		value, ok := v["value"]
		backend := false
		_, exist := v["backend"]
		if exist {
			backend = v["backend"].(bool)
		}
		if ok && !backend {
			result[k] = value
		}
	}
	return result
}
