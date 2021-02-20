package main

import (
	"ghscraper.htm/log"
	"ghscraper.htm/system"
)

func main() {
	if err := system.InitConfig(); err != nil {
		log.Fatal.Fatal("Failed to set up config from environment")
	}
	log.InitLogger()

	log.Info.Printf("Db Type: %s, Mode: %s\n", system.Cfg.DbType, system.Cfg.Mode)
}
