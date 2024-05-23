package query

import (
	"context"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"os"
	"strconv"
	"time"
)

func SetDatabaseProfilingLevel(client *mongo.Client, db string) error {
	slowms := os.Getenv("SLOW_MS")
	if slowms == "" {
		slowms = "0"
	}
	msInt, err := strconv.Atoi(slowms)
	if err != nil {
		return err
	}
	command := bson.D{
		{Key: "profile", Value: 1},
		{Key: "slowms", Value: msInt},
	}
	return client.Database(db).RunCommand(context.TODO(), command).Err()
}

func GetQueries(client *mongo.Client, db string) ([]map[string]interface{}, error) {
	queries := make([]map[string]interface{}, 0)

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	cur, err := client.Database(db).Collection("system.profile").Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}

	for cur.Next(ctx) {
		var result bson.D
		err := cur.Decode(&result)
		if err != nil {
			return nil, err
		}
		tem, err := bson.MarshalExtJSON(result, true, true)
		if err != nil {
			return nil, err
		}
		var mp map[string]interface{}
		err = json.Unmarshal(tem, &mp)
		if err != nil {
			return nil, err
		}
		queries = append(queries, mp)
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	err = cur.Close(ctx)
	if err != nil {
		return nil, err
	}
	return queries, nil
}
