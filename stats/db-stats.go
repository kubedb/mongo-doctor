package stats

import (
	"context"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/klog/v2"
	"kubedb.dev/mongo-doctor/database"
	"kubedb.dev/mongo-doctor/utils"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	shouldSkip = true
	summed     bson.M
	dir        = "all-stats"
)

func Run(client *mongo.Client) {
	sk := os.Getenv("SKIP")
	if sk == "false" {
		shouldSkip = false
	}

	utils.MakeDir(dir)
	collMap := database.ListCollectionsForAllDatabases(client)
	for db, collections := range collMap {
		if shouldSkip && utils.SkipDB(db) {
			continue
		}
		utils.MakeDir(filepath.Join(dir, db))
		dbRef := client.Database(db)
		databaseStats(dbRef)
		for _, coll := range collections {
			if shouldSkip && utils.SkipCollection(coll) {
				continue
			}
			collectionStats(dbRef, coll)
		}
	}
	admin := client.Database("admin")
	direct(admin, "serverStatus")
	direct(admin, "replSetGetStatus")
	direct(admin, "replSetGetConfig")
	direct(admin, "currentOp")

	// error : no such command
	//direct(admin, "getReplicationInfo")
	//direct(admin, "printSecondaryReplicationInfo")
	indentedData, err := json.MarshalIndent(summed, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	utils.WriteFile(dir, "_", indentedData)
	klog.Infof("sleep starts. You can run `kubectl cp demo/util:/app/all-stats /tmp/data` now.")
	time.Sleep(time.Minute * 10)
}

func databaseStats(db *mongo.Database) {
	cmd := bson.D{{"dbStats", 1}, {"scale", 1048576}}
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
		case "indexes", "objects", "collections":
			cur, found := summed[s]
			if !found {
				cur = int32(0)
				summed[s] = cur
			}
			summed[s] = cur.(int32) + obj.(int32)
		}

	}
}
