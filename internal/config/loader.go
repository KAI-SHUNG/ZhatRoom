package config

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

func LoadConfig[T ServerConfig | ClientConfig](
	configPath string,
	config *T) error {

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return fmt.Errorf("config file not found: %w", err)
		}
		return fmt.Errorf("error reading config file: %w", err)
	}
	if err := viper.Unmarshal(config); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}

	fmt.Printf("Config loaded successfully: %+v\n", config)

	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("Config file changed: %s\n", e.Name)
		temp := config
		if err := viper.Unmarshal(temp); err != nil {
			fmt.Printf("Error unmarshaling config: %v\n", err)
		} else {
			config = temp
			fmt.Printf("Config reloaded successfully\n")
		}
	})
	viper.WatchConfig()

	return nil
}
