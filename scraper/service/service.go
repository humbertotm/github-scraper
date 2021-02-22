package service

import (
	"fmt"

	"ghscraper.htm/domain"
	"ghscraper.htm/log"
	datastore "ghscraper.htm/scraper/data_store"
	externalapi "ghscraper.htm/scraper/external_api"
	"ghscraper.htm/system"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

// ScraperService defines the interface for a scraper entity
type ScraperService interface {
	Scrape() error
}

type scraperService struct {
	externalAPI externalapi.ExternalAPI
	dataStore   datastore.DataStore
}

var scraper ScraperService

// NewScraperService returns an instance of ScraperService
func NewScraperService(db neo4j.Driver, baseURL string) ScraperService {
	if scraper == nil {
		scraper = &scraperService{
			externalAPI: externalapi.NewExternalAPI(baseURL),
			dataStore:   datastore.NewDataStore(db),
		}
	}

	return scraper
}

// Scrape defines the general flow for scraping external API.
// It begins by scraping repos and its satellite users (owner and its respective
// followers and followees).
// Once repos are exhausted, it continues by scraping users in order by external_id
func (s *scraperService) Scrape() error {
	for s.externalAPI.ReqCount() <= system.MaxRequestPerHour() {
		if err := s.scrapeRepos(); err != nil {
			if err.Error() == "No more repos to scrape" {
				log.Info.Println("No more repos left to scrape. Continuing to scrape users")
			} else {
				return err
			}
		}

		if err := s.scrapeUsers(); err != nil {
			return err
		}
	}

	return nil
}

func (s *scraperService) scrapeRepos() error {
	startRepoID := s.dataStore.ReadMaxRepoExternalID()
	url := fmt.Sprintf("%s/repositories?since=%d", s.externalAPI.BaseURL(), startRepoID)

	repos, err := s.externalAPI.Get(url)

	if err != nil {
		log.Error.Println(err.Error())
		return err
	}

	if len(repos) == 0 {
		return fmt.Errorf("No more repos to scrape")
	}

	for _, repo := range repos {
		// First, store retrieved repo node
		if err := s.dataStore.WriteNode(domain.RepoLabel, repo); err != nil {
			log.Error.Println(err.Error())
			continue
		}

		// Store repo owner as a user node and create owner relationship
		var owner map[string]interface{}
		owner = repo["owner"].(map[string]interface{})
		if err := s.createOwner(owner, repo); err != nil {
			log.Error.Println(err.Error())
			continue
		}

		// Retrieve repo contributors, store them and create contributor relationship
		contributors, err := s.externalAPI.Get(repo["contributors_url"].(string))
		if err != nil {
			log.Error.Println(err.Error())
			continue
		}
		for _, contributor := range contributors {
			if err := s.createContributor(contributor, repo); err != nil {
				log.Error.Println(err.Error())
				continue
			}
		}

		// Retrieve owner followers, store them and create follows relationship
		followers, err := s.externalAPI.Get(owner["followers_url"].(string))
		if err != nil {
			log.Error.Println(err.Error())
			continue
		}
		for _, follower := range followers {
			if err := s.createFollower(owner, follower); err != nil {
				log.Error.Print(err.Error())
				continue
			}
		}

		// Retrieve owner following, store them and create follows relationship
		followed, err := s.externalAPI.Get(owner["following_url"].(string))
		if err != nil {
			log.Error.Println(err.Error())
			continue
		}
		for _, followee := range followed {
			if err := s.createFollower(owner, followee); err != nil {
				log.Error.Print(err.Error())
				continue
			}
		}
	}

	return nil
}

func (s *scraperService) scrapeUsers() error {
	minID := s.dataStore.ReadUserBookmark()
	url := fmt.Sprintf("%s/users?since=%d", s.externalAPI.BaseURL(), minID)

	users, err := s.externalAPI.Get(url)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		return fmt.Errorf("No more users to process")
	}

	for _, user := range users {
		// Store user if it is not stored yet
		if err := s.dataStore.WriteNode(domain.UserLabel, user); err != nil {
			log.Error.Println(err.Error())
			continue
		}

		// Retrieve user followers, store them and create follows relationship
		followers, err := s.externalAPI.Get(user["followers_url"].(string))
		if err != nil {
			log.Error.Println(err.Error())
			continue
		}
		for _, follower := range followers {
			if err := s.createFollower(user, follower); err != nil {
				log.Error.Print(err.Error())
				continue
			}
		}

		// Retrieve user following, store them and create follows relationship
		followed, err := s.externalAPI.Get(user["following_url"].(string))
		if err != nil {
			log.Error.Println(err.Error())
			continue
		}
		for _, followee := range followed {
			if err := s.createFollower(user, followee); err != nil {
				log.Error.Print(err.Error())
				continue
			}
		}
	}

	return nil
}

// Auxiliary functions dealing with the creation of nodes and relationships ensuring
// certain guarantees in terms of node/vertex creation order
func (s *scraperService) createOwner(owner, repo map[string]interface{}) error {
	if err := s.dataStore.WriteNode(domain.UserLabel, owner); err != nil {
		return err
	}

	startNodeData := domain.RelationshipNodeData{
		Label:      domain.UserLabel,
		MatchProp:  "username",
		ParamName:  "username",
		ParamValue: owner["login"].(string),
	}
	endNodeData := domain.RelationshipNodeData{
		Label:      domain.RepoLabel,
		MatchProp:  "name",
		ParamName:  "repo_name",
		ParamValue: repo["name"].(string),
	}

	if err := s.dataStore.WriteRelationship(startNodeData, endNodeData, domain.OwnerLabel); err != nil {
		return err
	}

	return nil
}

func (s *scraperService) createFollower(follower, followed map[string]interface{}) error {
	if err := s.dataStore.WriteNode(domain.UserLabel, follower); err != nil {
		return err
	}

	startNodeData := domain.RelationshipNodeData{
		Label:      domain.UserLabel,
		MatchProp:  "username",
		ParamName:  "follower_username",
		ParamValue: follower["login"].(string),
	}
	endNodeData := domain.RelationshipNodeData{
		Label:      domain.UserLabel,
		MatchProp:  "username",
		ParamName:  "followed_username",
		ParamValue: followed["login"].(string),
	}

	if err := s.dataStore.WriteRelationship(startNodeData, endNodeData, domain.FollowerLabel); err != nil {
		return err
	}

	return nil

}

func (s *scraperService) createContributor(contributor, repo map[string]interface{}) error {
	if err := s.dataStore.WriteNode(domain.UserLabel, contributor); err != nil {
		return err
	}

	startNodeData := domain.RelationshipNodeData{
		Label:      domain.UserLabel,
		MatchProp:  "username",
		ParamName:  "username",
		ParamValue: contributor["login"].(string),
	}
	endNodeData := domain.RelationshipNodeData{
		Label:      domain.RepoLabel,
		MatchProp:  "name",
		ParamName:  "repo_name",
		ParamValue: repo["name"].(string),
	}

	if err := s.dataStore.WriteRelationship(startNodeData, endNodeData, domain.ContributorLabel); err != nil {
		return err
	}

	return nil

}
