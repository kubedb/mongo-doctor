package diffchecker

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/klog/v2"
	"kubedb.dev/mongo-doctor/database"
	"kubedb.dev/mongo-doctor/utils"
	"log"
	"strings"
)

func Run(kubedbClient, atlasClient *mongo.Client) {
	atlasCollMap := database.ListCollectionsForAllDatabases(atlasClient)
	kubedbCollMap := database.ListCollectionsForAllDatabases(kubedbClient)
	for db, collections := range atlasCollMap {
		if utils.SkipDB(db) {
			continue
		}
		klog.Infof("------------------------ %v ---------------------- \n", db)
		atlasDBRef := atlasClient.Database(db)
		if _, exists := kubedbCollMap[db]; !exists {
			klog.Infof("%s db exists in atlas but doesn't exist in kubedb MongoDB", db)
			continue
		}
		kubedbdbDBRef := kubedbClient.Database(db)
		for _, coll := range collections {
			if utils.SkipCollection(coll) {
				continue
			}
			atlasStat := collectionStats(atlasDBRef, coll)
			if !utils.ContainsString(kubedbCollMap[db], coll) {
				klog.Infof("collection %s in db %s exists in atlas but doesn't exist in kubedb MongoDB", coll, db)
			}
			kubedbStat := collectionStats(kubedbdbDBRef, coll)
			compare(coll, atlasStat, kubedbStat)
		}
	}
}

func collectionStats(db *mongo.Database, coll string) bson.M {
	cmd := bson.D{{"collStats", coll}, {"scale", 1048576}}
	var result bson.M
	err := db.RunCommand(context.TODO(), cmd).Decode(&result)
	if err != nil {
		if strings.Contains(err.Error(), "is a view, not a collection") {
			klog.Infoln(err.Error())
			return nil
		} else {
			log.Fatal(err)
		}
	}
	output := make(bson.M)
	for s, obj := range result {
		switch s {
		case "nindexes", "indexSizes":
			output[s] = obj
		}
	}
	return output
}

func compare(coll string, aStats, kStats bson.M) {
	aIndexCount := aStats["nindexes"].(int32)
	kIndexCount := kStats["nindexes"].(int32)
	if aIndexCount != kIndexCount {
		klog.Infof("indexCount not matched for collection %s. In atlas %v VS In kubedb %v \n", coll, aIndexCount, kIndexCount)
		klog.Infof("atlasIndexes: %v kubedbIndexes: %v\n", aStats["indexSizes"], kStats["indexSizes"])
	} else {
		klog.Infof("indexCount matched for collection %s. count=%v \n", coll, aIndexCount)
	}
}
