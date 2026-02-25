package stats

import (
	"context"
	"encoding/json"
	"log"
	"path/filepath"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"kubedb.dev/mongo-doctor/database"
	"kubedb.dev/mongo-doctor/utils"
)

func Collect(client *mongo.Client, dir string, collectRepl bool) {
	utils.MakeDir(dir)
	admin := client.Database("admin")
	if !collectRepl {
		direct(admin, "serverStatus", dir)
		direct(admin, "currentOp", dir)
		// direct(admin, "sh.status", dir). Not possible to run user utility like this. Equivalent: listShards, getShardMap,balancerStatus
		return
	}
	collMap := database.ListCollectionsForAllDatabases(client)
	for db, _ := range collMap {
		if shouldSkip && utils.SkipDB(db) {
			continue
		}
		utils.MakeDir(filepath.Join(dir, db))
		dbRef := client.Database(db)
		databaseStats(dbRef, dir)
		//for _, coll := range collections {
		//	if shouldSkip && utils.SkipCollection(coll) {
		//		continue
		//	}
		//	collectionStats(dbRef, coll, dir)
		//}
	}

	direct(admin, "replSetGetStatus", dir)
	direct(admin, "replSetGetConfig", dir)

	// error : no such command
	//direct(admin, "getReplicationInfo")
	//direct(admin, "printSecondaryReplicationInfo")
	indentedData, err := json.MarshalIndent(summed, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	utils.WriteFile(dir, "_", indentedData)
	summed = nil
}

func databaseStats(db *mongo.Database, dir string) {
	cmd := bson.D{{"dbStats", 1}, {"scale", 1024 * 1024}}
	var result bson.M
	err := db.RunCommand(context.TODO(), cmd).Decode(&result)
	if err != nil {

	}
	indentedData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	utils.WriteFile(filepath.Join(dir, db.Name()), "_", indentedData)
	sumUp(result)
}

/*
	{
	  "collections": 5,
	  "dataSize": 23213.150283813477,
	  "fsTotalSize": 271057.07421875,
	  "fsUsedSize": 94407.203125,
	  "indexSize": 14814.77734375,
	  "indexes": 4,
	  "objects": 529124951,
	  "storageSize": 14797.8671875,
	  "totalSize": 29612.64453125,
	},
*/
func sumUp(result bson.M) {
	if summed == nil {
		summed = make(bson.M)
	}
	for s, obj := range result {
		switch s {
		case "dataSize", "fsTotalSize", "fsUsedSize", "indexSize", "storageSize", "totalSize":
			cur, found := summed[s]
			if !found {
				cur = float64(0)
				summed[s] = cur
			}
			summed[s] = cur.(float64) + obj.(float64)
		case "indexes", "collections":
			cur, found := summed[s]
			if !found {
				cur = int32(0)
				summed[s] = cur
			}
			summed[s] = cur.(int32) + obj.(int32)
		case "objects":
			cur, found := summed[s]
			if !found {
				cur = int64(0)
				summed[s] = cur
			}
			inInt := obj.(int32)
			summed[s] = cur.(int64) + int64(inInt)
		}

	}
}
