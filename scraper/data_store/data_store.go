package datastore

import (
	"fmt"

	"ghscraper.htm/domain"
	"ghscraper.htm/log"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

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

// DataStore defines the interface for the persistent storage layer
type DataStore interface {
	WriteNode(nodeLabel string, nodeProperties map[string]interface{}) error
	WriteRelationship(startNode, endNode domain.RelationshipNodeData, relationshipLabel string) error
	ReadMaxRepoExternalID() int
	ReadUserBookmark() int
	UpdateUserBookmark(external_id interface{}) error
}

type dataStore struct {
	db neo4j.Driver
}

var store DataStore

// NewDataStore returns an instance implementing DataStore
func NewDataStore(db neo4j.Driver) DataStore {
	if store == nil {
		store = &dataStore{db: db}
	}

	return store
}

// WriteNode inserts a node with the specified label and properties if it does not exist
func (dw *dataStore) WriteNode(nodeLabel string, nodeProperties map[string]interface{}) error {
	properties := dw.getPropertiesString(nodeLabel)
	paramsMap := dw.getParamsMap(nodeLabel, nodeProperties)
	query := fmt.Sprintf("MERGE (n:%s %s)", nodeLabel, properties)
	log.Info.Printf("[WRITE_NODE] Executing query %s\n", query)

	return dw.write(query, paramsMap)
}

// WriteRelationship inserts a vertex specifying a relationship between two nodes
func (dw *dataStore) WriteRelationship(startNode, endNode domain.RelationshipNodeData, relationshipLabel string) error {
	query := dw.buildRelationshipQuery(startNode, endNode, relationshipLabel)
	paramsMap := map[string]interface{}{
		startNode.ParamName: startNode.ParamValue,
		endNode.ParamName:   endNode.ParamValue,
	}
	log.Info.Printf("[WRITE_RELATIONSHIP] Executing query %s\n", query)

	return dw.write(query, paramsMap)
}

// ReadMaxRepoExternalID returns the max repo ID for the scraping process to pick up where
// it left off when scraping by repos
func (dw *dataStore) ReadMaxRepoExternalID() int {
	query := "MATCH (r:Repo) RETURN r.external_id ORDER BY r.external_id DESC LIMIT 1"
	data, err := dw.query(query, nil)
	if err != nil {
		return 0
	}

	return int(data.Values[0].(float64))
}

// ReadUserBookmark returns the id of the bookmarked user node for the scraping process to
// pick up where it left off when scraping by users
func (dw *dataStore) ReadUserBookmark() int {
	query := "MATCH (b:UserBookmark) RETURN b.external_id LIMIT 1"
	data, err := dw.query(query, nil)
	if err != nil {
		return 0
	}

	return int(data.Values[0].(int64))
}

// UpdateUserBookmark updates the bookmark with last user external_id it retrieved from
// external API
func (dw *dataStore) UpdateUserBookmark(external_id interface{}) error {
	query := "MERGE (b:UserBookmark) SET b.external_id = $external_id"

	return dw.write(query, map[string]interface{}{"external_id": external_id})
}

func (dw *dataStore) write(query string, params map[string]interface{}) error {
	session := dw.db.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		_, err := tx.Run(query, params)

		if err != nil {
			return nil, err
		}

		// Don't need the value, just to know that transaction was successful
		return nil, nil
	})

	return err
}

func (dw *dataStore) query(query string, params map[string]interface{}) (neo4j.Record, error) {
	session := dw.db.NewSession(neo4j.SessionConfig{})
	defer session.Close()
	data, err := session.Run(query, params)
	if err != nil {
		return neo4j.Record{}, err
	}

	if !data.Next() {
		return neo4j.Record{}, fmt.Errorf("No record found")
	}

	return *data.Record(), nil
}

func (dw *dataStore) getPropertiesString(nodeLabel string) string {
	switch nodeLabel {
	case domain.UserLabel:
		return userProperties
	case domain.RepoLabel:
		return repoProperties
	default:
		return ""
	}
}

func (dw *dataStore) getParamsMap(nodeLabel string, props map[string]interface{}) map[string]interface{} {
	switch nodeLabel {
	case domain.UserLabel:
		return map[string]interface{}{
			"username":      props["login"],
			"external_id":   props["id"],
			"user_url":      props["url"],
			"followers_url": props["followers_url"],
			"following_url": props["following_url"],
			"repos_url":     props["repos_url"],
			"type":          props["type"],
			"site_admin":    props["site_admin"],
			"created_at":    neo4j.Time{},
			"updated_at":    neo4j.Time{},
		}
	case domain.RepoLabel:
		return map[string]interface{}{
			"external_id":      props["id"],
			"name":             props["name"],
			"full_name":        props["full_name"],
			"html_url":         props["html_url"],
			"url":              props["url"],
			"contributors_url": props["contributors_url"],
			"issues_url":       props["issues_url"],
			"languages_url":    props["languages_url"],
			"created_at":       neo4j.Time{},
			"updated_at":       neo4j.Time{},
		}
	default:
		return nil
	}
}

func (dw *dataStore) buildRelationshipQuery(startNode, endNode domain.RelationshipNodeData, relationshipLabel string) string {
	return fmt.Sprintf(
		"MATCH (s:%s {%s: $%s}) MATCH (e:%s {%s: $%s}) MERGE (s)-[%s]->(e)",
		startNode.Label,
		startNode.MatchProp,
		startNode.ParamName,
		endNode.Label,
		endNode.MatchProp,
		endNode.ParamName,
		relationshipLabel,
	)
}
