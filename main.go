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
		if err.Error() == "Rate limit exceeded" {
			log.Info.Println("Rate limit exceeded. Terminating process.")
			os.Exit(0)
		}
		log.Error.Println(err.Error())
		os.Exit(1)
	}

	log.Info.Println("You're done!")
	os.Exit(0)
}
