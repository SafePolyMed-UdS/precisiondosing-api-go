package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"precisiondosing-api-go/cfg"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoConnection struct {
	Client     *mongo.Client
	Database   string
	Collection string
}

func New(dbConfig cfg.MongoConfig) (*MongoConnection, error) {
	clientOptions := options.Client().ApplyURI(dbConfig.URI)
	clientOptions.SetMaxPoolSize(dbConfig.MaxPoolSize)
	clientOptions.SetMinPoolSize(dbConfig.MinPoolSize)
	clientOptions.SetMaxConnIdleTime(dbConfig.MaxIdletime)

	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to %s: %w", dbConfig.URI, err)
	}

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		return nil, fmt.Errorf("cannot ping %s: %w", dbConfig.URI, err)
	}

	// Check if database and collection exist
	collection := client.Database(dbConfig.Database).Collection(dbConfig.Collection)
	var testFetch bson.M
	err = collection.FindOne(context.TODO(), bson.M{}).Decode(&testFetch)
	if err != nil || testFetch == nil {
		return nil, fmt.Errorf("cannot find collection %s in database %s", dbConfig.Collection, dbConfig.Database)
	}
	result := &MongoConnection{
		Client:     client,
		Database:   dbConfig.Database,
		Collection: dbConfig.Collection,
	}

	return result, nil
}

// FetchIndividual fetches an individual from the database.
// If no individual is found, nil is returned.
func (m *MongoConnection) FetchIndividual(
	population *string,
	gender string, age, height, weight int,
) (json.RawMessage, error) {
	eth, err := mapEthnicity(population)
	if err != nil {
		return nil, fmt.Errorf("error mapping ethnicity: %w", err)
	}

	gender = strings.ToUpper(gender)
	query := bson.D{
		{Key: "population", Value: eth},
		{Key: "age", Value: age},
		{Key: "weight", Value: weight},
		{Key: "height", Value: height},
		{Key: "gender", Value: gender},
	}

	collection := m.Client.Database(m.Database).Collection(m.Collection)
	var result bson.M
	err = collection.FindOne(context.TODO(), query).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("error querying individual: %w", err)
	}

	if result == nil {
		return nil, nil
	}

	payload, ok := result["json"]
	if !ok {
		return nil, errors.New("error: 'json' key not found in result")
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling 'json' field to RawMessage: %w", err)
	}

	return jsonData, nil
}

func mapEthnicity(ethnicity *string) (string, error) {
	// Default value if ethnicity is nil
	defaultPopulation := "European_ICRP_2002"

	// Ethnicity-to-population mapping
	mapping := map[string]string{
		"european":       "European_ICRP_2002",
		"white american": "WhiteAmerican_NHANES_1997",
		"black american": "BlackAmerican_NHANES_1997",
		"mexican":        "MexicanAmericanWhite_NHANES_1997",
		"asian":          "Asian_Tanaka_1996",
		"japanese":       "Japanese_Population",
		"other":          defaultPopulation,
		"unknown":        defaultPopulation,
	}

	// If ethnicity is nil, return the default value
	if ethnicity == nil {
		return defaultPopulation, nil
	}

	// Normalize input by trimming spaces and converting to lowercase
	normalized := strings.ToLower(strings.TrimSpace(*ethnicity))

	// Return the mapped value if found
	if mapped, exists := mapping[normalized]; exists {
		return mapped, nil
	}

	// Return an error if mapping fails
	return "", errors.New("unknown ethnicity: " + *ethnicity)
}
