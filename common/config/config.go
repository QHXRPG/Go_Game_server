package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config 反引号标记字段的元数据
// 通常用于序列化和反序列化（如 JSON、XML、YAML 等）时指定字段的映射名称。
type Config struct {
	Log        LogConf                 `mapstructure:"log"`
	Port       int                     `mapstructure:"port"`
	WsPort     int                     `mapstructure:"wsPort"`
	MetricPort int                     `mapstructure:"metricPort"`
	HttpPort   int                     `mapstructure:"httpPort"`
	AppName    string                  `mapstructure:"appName"`
	Database   Database                `mapstructure:"db"`
	Jwt        JwtConf                 `mapstructure:"jwt"`
	Grpc       GrpcConf                `mapstructure:"grpc"`
	Etcd       EtcdConf                `mapstructure:"etcd"`
	Domain     map[string]Domain       `mapstructure:"domain"`
	Services   map[string]ServicesConf `mapstructure:"services"`
}
type ServicesConf struct {
	Id         string `mapstructure:"id"`
	ClientHost string `mapstructure:"clientHost"`
	ClientPort int    `mapstructure:"clientPort"`
}
type Domain struct {
	Name        string `mapstructure:"name"`
	LoadBalance bool   `mapstructure:"loadBalance"`
}
type JwtConf struct {
	Secret string `mapstructure:"secret"`
	Exp    int64  `mapstructure:"exp"`
}
type LogConf struct {
	Level string `mapstructure:"level"`
}

// Database 数据库配置
type Database struct {
	MongoConf MongoConf `mapstructure:"mongo"`
	RedisConf RedisConf `mapstructure:"redis"`
}
type MongoConf struct {
	Url         string `mapstructure:"url"`
	Db          string `mapstructure:"db"`
	UserName    string `mapstructure:"userName"`
	Password    string `mapstructure:"password"`
	MinPoolSize int    `mapstructure:"minPoolSize"`
	MaxPoolSize int    `mapstructure:"maxPoolSize"`
}
type RedisConf struct {
	Addr         string   `mapstructure:"addr"`
	ClusterAddrs []string `mapstructure:"clusterAddrs"`
	Password     string   `mapstructure:"password"`
	PoolSize     int      `mapstructure:"poolSize"`
	MinIdleConns int      `mapstructure:"minIdleConns"`
	Host         string   `mapstructure:"host"`
	Port         int      `mapstructure:"port"`
}
type EtcdConf struct {
	Addrs       []string       `mapstructure:"addrs"`
	RWTimeout   int            `mapstructure:"rwTimeout"`
	DialTimeout int            `mapstructure:"dialTimeout"`
	Register    RegisterServer `mapstructure:"register"`
}
type RegisterServer struct {
	Addr    string `mapstructure:"addr"`
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Weight  int    `mapstructure:"weight"`
	Ttl     int64  `mapstructure:"ttl"` //租约时长
}
type GrpcConf struct {
	Addr string `mapstructure:"addr"`
}

// Conf 声明一个指向Config结构体的指针
var Conf *Config

// InitConfig 加载配置
// 这个函数用于初始化配置文件，并将其内容解析到全局变量 Conf 中。
func InitConfig(confFile string) {
	Conf = new(Config)        // 创建一个新的 Config 对象
	v := viper.New()          // 创建一个新的 viper 实例，用于读取配置文件。
	v.SetConfigFile(confFile) // 设置配置文件的路径。

	// 监听配置文件的修改
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("config file changed:", e.Name)
	})

	// 尝试读取配置文件。
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	// 将读取到的配置文件内容解析(反序列化)到全局变量 Conf 中。
	err = v.Unmarshal(&Conf)
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
}
