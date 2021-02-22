package db

import (
	"ghscraper.htm/system"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

func InitDb() (neo4j.Driver, error) {
	return neo4j.NewDriver(system.Cfg.DbURL, neo4j.BasicAuth(system.Cfg.DbUsername, system.Cfg.DbPassword, ""))
}
