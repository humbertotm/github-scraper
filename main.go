package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ghscraper.htm/log"
	"ghscraper.htm/system"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

const maxRequests = 5
const exceededRequestLimitStatus = http.StatusForbidden
const repoProperties = `{
external_id: $external_id,
name: $name,
full_name: $full_name,
html_url: $html_url,
url: $url,
contributors_url: $contributors_url,
issues_url: $issues_url,
languages_url: $languages_url,
created_at: $created_at,
updated_at: $updated_at
}`
const userProperties = `{
username: $username,
external_id: $external_id,
user_url: $user_url,
followers_url: $followers_url,
following_url: $following_url,
repos_url: $repos_url,
type: $type,
site_admin: $site_admin,
created_at: $created_at,
updated_at: $updated_at
}`

type User struct {
	Username     string `json:"login"`
	ExternalID   int    `json:"id"`
	UserURL      string `json:"url"`
	FollowersURL string `json:"followers_url"`
	FollowingURL string `json:"following_url"`
	ReposURL     string `json:"repos_url"`
	Type         string `json:"type"`
	SiteAdmin    bool   `json:"site_admin"`
	CreatedAt    *time.Time
	UpdatedAt    *time.Time
}

type Repo struct {
	ExternalID      int    `json:"id"`
	Name            string `json:"name"`
	FullName        string `json:"full_name"`
	Owner           *User  `json:"owner"`
	HTMLUrl         string `json:"html_url"`
	URL             string `json:"url"`
	ContributorsURL string `json:"contributors_url"`
	IssuesURL       string `json:"issues_url"`
	LanguagesURL    string `json:"languages_url"`
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
}

func main() {
	if err := system.InitConfig(); err != nil {
		log.Fatal.Fatal("Failed to set up config from environment")
	}
	log.InitLogger()

	driver, err := InitDbDriver()
	if err != nil {
		log.Fatal.Fatal(err.Error())
	}
	defer driver.Close()

	if err := Scrape(driver); err != nil {
		log.Fatal.Fatal(err.Error())
	}

	log.Info.Println("Done!")
}

func InitDbDriver() (neo4j.Driver, error) {
	return neo4j.NewDriver("neo4j://localhost:7687", neo4j.BasicAuth("", "", ""))
}

func Scrape(driver neo4j.Driver) error {
	minRepoID := 0
	requestCount := 0
	for requestCount <= maxRequests {
		repos, err := GetRepos(&requestCount, minRepoID)
		if err != nil {
			log.Error.Println(err.Error())
			return err
		}

		for _, repo := range repos {
			if err := CreateRepo(driver, repo); err != nil {
				log.Error.Print(err.Error())
				continue
			}
			var owner map[string]interface{}
			owner = repo["owner"].(map[string]interface{})
			if err := CreateOwner(driver, owner, repo); err != nil {
				log.Error.Print(err.Error())
				continue
			}

			followers, err := GetFollowers(&requestCount, owner)
			if err != nil {
				log.Error.Print(err.Error())
				continue
			}
			for _, follower := range followers {
				if err := CreateFollower(driver, owner, follower); err != nil {
					log.Error.Print(err.Error())
					continue
				}
			}

			followings, err := GetFollowing(&requestCount, owner)
			if err != nil {
				log.Error.Print(err.Error())
				continue
			}
			for _, following := range followings {
				if err := CreateFollower(driver, following, owner); err != nil {
					log.Error.Print(err.Error())
					continue
				}
			}

			contributors, err := GetContributors(&requestCount, repo)
			if err != nil {
				log.Error.Print(err.Error())
				continue
			}
			for _, contributor := range contributors {
				if err := CreateContributor(driver, repo, contributor); err != nil {
					log.Error.Print(err.Error())
					continue
				}
			}
		}

	}
	return nil
}

func GetRepos(reqCounter *int, minID int) ([]map[string]interface{}, error) {
	client := &http.Client{}
	var respJSON []map[string]interface{}

	req, err := http.NewRequest("GET", "https://api.github.com/repositories", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	*reqCounter = *reqCounter + 1
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Request limit exceeded")
	}

	if err := json.NewDecoder(resp.Body).Decode(&respJSON); err != nil {
		return nil, err
	}

	return respJSON, nil
}

func GetFollowers(reqCounter *int, user map[string]interface{}) ([]map[string]interface{}, error) {
	client := &http.Client{}
	var respJSON []map[string]interface{}

	req, err := http.NewRequest("GET", user["followers_url"].(string), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	*reqCounter = *reqCounter + 1
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Request limit exceeded")
	}

	if err := json.NewDecoder(resp.Body).Decode(&respJSON); err != nil {
		return nil, err
	}

	return respJSON, nil
}

func GetFollowing(reqCounter *int, user map[string]interface{}) ([]map[string]interface{}, error) {
	client := &http.Client{}
	var respJSON []map[string]interface{}

	req, err := http.NewRequest("GET", user["following_url"].(string), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	*reqCounter = *reqCounter + 1
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Request limit exceeded")
	}

	if err := json.NewDecoder(resp.Body).Decode(&respJSON); err != nil {
		return nil, err
	}

	return respJSON, nil
}

func GetContributors(reqCounter *int, repo map[string]interface{}) ([]map[string]interface{}, error) {
	client := &http.Client{}
	var respJSON []map[string]interface{}

	req, err := http.NewRequest("GET", repo["contributors_url"].(string), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	*reqCounter = *reqCounter + 1
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Request limit exceeded")
	}

	if err := json.NewDecoder(resp.Body).Decode(&respJSON); err != nil {
		return nil, err
	}

	return respJSON, nil
}

func CreateRepo(driver neo4j.Driver, repo map[string]interface{}) error {
	log.Info.Printf("Creating/Merging repo with name: %s\n", repo["name"])

	query := fmt.Sprintf("MERGE (r:Repo %s)", repoProperties)
	propertiesMap := getRepoPropertiesMap(repo)
	return WriteItem(driver, query, propertiesMap)
}

func CreateOwner(driver neo4j.Driver, owner, repo map[string]interface{}) error {
	if err := CreateUser(driver, owner); err != nil {
		return err
	}

	if err := CreateOwnerRelationship(driver, owner, repo); err != nil {
		return err
	}

	return nil
}

func CreateFollower(driver neo4j.Driver, user, follower map[string]interface{}) error {
	if err := CreateUser(driver, follower); err != nil {
		return err
	}

	if err := CreateFollowerRelationship(driver, user, follower); err != nil {
		return err
	}

	return nil
}

func CreateContributor(driver neo4j.Driver, repo, contributor map[string]interface{}) error {
	if err := CreateUser(driver, contributor); err != nil {
		return err
	}

	if err := CreateContributorRelationship(driver, repo, contributor); err != nil {
		return err
	}

	return nil
}

func CreateUser(driver neo4j.Driver, user map[string]interface{}) error {
	log.Info.Printf("Creating/Merging user with name: %s\n", user["login"].(string))

	query := fmt.Sprintf("MERGE (r:User %s)", userProperties)
	propertiesMap := getUserPropertiesMap(user)
	return WriteItem(driver, query, propertiesMap)
}

func CreateOwnerRelationship(driver neo4j.Driver, owner, repo map[string]interface{}) error {
	log.Info.Printf("Creating owns relationship for user %s, repo %s\n", owner["login"], repo["name"])

	query := "MATCH (u:User {username: $username}) MATCH (r:Repo {name: $repo_name}) MERGE (u)-[:OWNS]->(r)"
	propertiesMap := map[string]interface{}{
		"username":  owner["login"],
		"repo_name": repo["name"],
	}

	return WriteItem(driver, query, propertiesMap)
}

func CreateFollowerRelationship(driver neo4j.Driver, user, follower map[string]interface{}) error {
	log.Info.Printf("Creating follows relationship for user %s, follower %s\n", user["login"], follower["login"])

	query := "MATCH (u:User {username: $username}) MATCH (f:User {username: $follower_name}) MERGE (f)-[:FOLLOWS]->(u)"
	propertiesMap := map[string]interface{}{
		"username":      user["login"],
		"follower_name": follower["login"],
	}

	return WriteItem(driver, query, propertiesMap)
}

func CreateContributorRelationship(driver neo4j.Driver, repo, contributor map[string]interface{}) error {
	log.Info.Printf("Creating contributor relationship for repo %s, contributor %s\n", repo["name"], contributor["login"])

	query := "MATCH (u:User {username: $username}) MATCH (r:Repo {name: $repo_name}) MERGE (u)-[:CONTRIBUTOR]->(r)"
	propertiesMap := map[string]interface{}{
		"username":  contributor["login"],
		"repo_name": repo["name"],
	}

	return WriteItem(driver, query, propertiesMap)
}

// func CreateFollowedRelationship(driver neo4j.Driver, user, follower map[string]interface{}) error {
// 	log.Info.Printf("Creating follows relationship for user %s, follower %s\n", user["login"], follower["login"])

// 	query := "MATCH (u:User {username: $username}) MATCH (f:User {username: $follower_name}) MERGE (f)-[:FOLLOWS]->(u)"
// 	propertiesMap := map[string]interface{}{
// 		"username":  user["login"],
// 		"follower_name": follower["login"],
// 	}

// 	return WriteItem(driver, query, propertiesMap)
// }

func WriteItem(driver neo4j.Driver, query string, item map[string]interface{}) error {
	session := driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	_, err := session.WriteTransaction(insertItemFn(query, item))

	return err
}

func insertItemFn(queryStr string, item map[string]interface{}) func(neo4j.Transaction) (interface{}, error) {
	return func(tx neo4j.Transaction) (interface{}, error) {
		_, err := tx.Run(queryStr, item)
		// In face of driver native errors, make sure to return them directly.
		// Depending on the error, the driver may try to execute the function again.
		if err != nil {
			return nil, err
		}

		return nil, nil
	}
}

func getRepoPropertiesMap(repo map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"external_id":      repo["id"],
		"name":             repo["name"],
		"full_name":        repo["full_name"],
		"html_url":         repo["html_url"],
		"url":              repo["url"],
		"contributors_url": repo["contributors_url"],
		"issues_url":       repo["issues_url"],
		"languages_url":    repo["languages_url"],
		"created_at":       neo4j.Time{},
		"updated_at":       neo4j.Time{},
	}
}

func getUserPropertiesMap(user map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"username":      user["login"],
		"external_id":   user["id"],
		"user_url":      user["url"],
		"followers_url": user["followers_url"],
		"following_url": user["following_url"],
		"repos_url":     user["repos_url"],
		"type":          user["type"],
		"site_admin":    user["site_admin"],
		"created_at":    neo4j.Time{},
		"updated_at":    neo4j.Time{},
	}
}
