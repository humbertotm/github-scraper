package main

import (
	"os"

	"ghscraper.htm/db"
	"ghscraper.htm/log"
	"ghscraper.htm/scraper/service"
	"ghscraper.htm/system"
)

func main() {
	if err := system.InitConfig(); err != nil {
		log.Error.Fatal("Failed to set up config from environment")
	}
	log.InitLogger()

	db, err := db.InitDb()
	if err != nil {
		log.Fatal.Fatal(err.Error())
	}
	defer db.Close()

	ghScraperService := service.NewScraperService(db, system.Cfg.GithubBaseURL)

	if err := ghScraperService.Scrape(); err != nil {
		log.Error.Println(err.Error())
		os.Exit(1)
	}

	log.Info.Println("You're done! Come back in an hour once your request rate limit has been reset")
	os.Exit(0)
}
