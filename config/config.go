package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"sync"
)

type Read struct {
	Mysql struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"mysql"`
	Mode struct {
		Rows int64 `yaml:"rows"` // 当差异大于这个数值时,则报警
	} `yaml:"mode"`
}

// Config represents the configuration structure
type Config struct {
	Database struct {
		Driver string `yaml:"driver"`
		Source string `yaml:"source"`
	} `yaml:"database"`

	Read Read `yaml:"read"`

	FeiShu struct {
		App struct {
			Id     string `yaml:"id"`
			Secret string `yaml:"secret"`
		} `yaml:"app"`
		Drive struct {
			BaseId  string `yaml:"base_id"`
			TableId string `yaml:"table_id"`
		} `yaml:"drive"`
	} `yaml:"feishu"`
}

// GlobalConfig 存储全局配置
var GlobalConfig *Config
var configOnce sync.Once

// GetConfig 获取全局配置，确保配置只加载一次
func GetConfig() *Config {
	configOnce.Do(func() {
		var err error
		GlobalConfig, err = ReadConfig("config.yaml")
		if err != nil {
			fmt.Println("Error loading config:", err)
			os.Exit(1) // 配置加载失败时退出程序
		}
	})
	return GlobalConfig
}

func ReadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config)

	if err != nil {
		return nil, err
	}
	return &config, nil
}
