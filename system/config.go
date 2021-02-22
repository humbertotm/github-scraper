package system

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

var Cfg EnvConfig

type EnvConfig struct {
	Mode           string `envconfig:"mode"`
	DbURL          string `envconfig:"db_url"`
	DbUsername     string `envconfig:"db_username"`
	DbPassword     string `envconfig:"db_password"`
	GithubBaseURL  string `envconfig:"github_base_url"`
	LogFile        string `envconfig:"log_file"`
	BasicAuthToken string `envconfig:"basic_auth_token"`
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
	return Cfg.Mode == "dev"
}

func MaxRequestPerHour() int {
	if Cfg.BasicAuthToken != "" {
		return 5000
	}

	return 60
}
