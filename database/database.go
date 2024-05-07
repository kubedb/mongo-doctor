package database

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/klog/v2"
	"strings"
)

func ListDatabases(client *mongo.Client) []string {
	dbs, err := client.ListDatabaseNames(context.TODO(), bson.D{})
	if err != nil {
		panic(err)
	}
	return dbs
}

func ListCollectionsForSpecificDatabase(client *mongo.Client, database string) []string {
	db := client.Database(database)
	names, err := db.ListCollectionNames(context.Background(), bson.D{})
	if err != nil {
		panic(err)
	}
	return names
}

func ListCollectionsForAllDatabases(client *mongo.Client) map[string][]string {
	dbs := ListDatabases(client)
	mp := make(map[string][]string)
	for _, d := range dbs {
		db := client.Database(d)
		names, err := db.ListCollectionNames(context.Background(), bson.D{})
		if err != nil {
			panic(err)
		}
		mp[d] = names
	}
	return mp
}

func DBStats(ctx context.Context, client *mongo.Client, db string) (map[string]interface{}, error) {
	dbStats := make(map[string]interface{})
	err := client.Database(db).RunCommand(ctx, bson.D{{Key: "dbStats", Value: 1}}).Decode(&dbStats)
	return dbStats, err
}

// GetPrimaryAndSecondaries return the hosts. 0th element is primary, others are secondary
func GetPrimaryAndSecondaries(ctx context.Context, client *mongo.Client) ([]string, error) {
	adminDB := client.Database("admin")
	var result bson.M
	err := adminDB.RunCommand(ctx, bson.D{{"replSetGetStatus", 1}}).Decode(&result)
	if err != nil {
		klog.Infoln("Error running rs.status():", err)
		return nil, err
	}

	members, ok := result["members"].(primitive.A)
	if !ok {
		klog.Infoln("Error parsing members array")
		return nil, err
	}

	primary := ""
	var secondaries []string
	for _, member := range members {
		memberInfo, ok := member.(primitive.M)
		if !ok {
			klog.Infoln("Error parsing member information")
			continue
		}

		// Check member state
		state := memberInfo["state"].(int32)
		if state == 1 {
			primary = memberInfo["name"].(string)
		} else if state == 2 {
			secondaries = append(secondaries, memberInfo["name"].(string))
		}
	}
	var ret []string
	ret = append(ret, strings.Split(primary, ".")[0])
	for _, secondary := range secondaries {
		ret = append(ret, strings.Split(secondary, ".")[0])
	}
	return ret, nil
}
