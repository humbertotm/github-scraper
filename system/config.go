package system

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

var Cfg EnvConfig

type EnvConfig struct {
	Mode    string `envconfig:"mode"`
	LogFile string `envconfig:"log_file"`
	DbType  string `envconfig:"db_type"`
}

func InitConfig() error {
	if err := godotenv.Load(); err != nil {
		return err
	}

	if err := envconfig.Process("", &Cfg); err != nil {
		return err
	}

	return nil
}

func IsDev() bool {
	fmt.Printf("cfg mode: %s\n", Cfg.Mode)
	fmt.Printf("cfg db type: %s\n", Cfg.DbType)
	return Cfg.Mode == "dev"
}
