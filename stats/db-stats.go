package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/klog/v2"
	kubedb "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	"kubedb.dev/mongo-doctor/database"
	"kubedb.dev/mongo-doctor/k8s"
	"kubedb.dev/mongo-doctor/mongoclient"
	"kubedb.dev/mongo-doctor/utils"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	mg       *kubedb.MongoDB
	password string

	shouldSkip = true
	summed     bson.M
	Dir        = "all-stats"

	primaryPod   string
	secondaryOne string
	secondaryTwo string
	err          error
)

func Run(client *mongo.Client) {
	sk := os.Getenv("SKIP")
	if sk == "false" {
		shouldSkip = false
	}

	mg, err = mongoclient.GetMongoDB()
	if err != nil {
		klog.Fatal(err)
	}

	secret, err := mongoclient.GetSecret(mg.Spec.AuthSecret.Name, mg.Namespace)
	if err != nil {
		klog.Fatal(err)
	}
	password = string(secret.Data["password"])

	hosts, err := database.GetPrimaryAndSecondaries(context.TODO(), client)
	if err != nil {
		_ = fmt.Errorf("error while getting primary and secondaries %v", err)
		return
	}
	primaryPod = hosts[0]
	secondaryOne = hosts[1]
	secondaryTwo = hosts[2]

	klog.Infof("Primary %v and Secondaries found! %v %v \n", primaryPod, secondaryOne, secondaryTwo)

	start := time.Now()
	klog.Infof("STATs starts at %v \n", start)

	Collect(client, Dir+"/"+primaryPod)

	tunnelOne, err := mongoclient.TunnelToDBPod(k8s.GetRESTConfig(), mg.Namespace, secondaryOne)
	if err != nil {
		_ = fmt.Errorf("tunnel creation failed for %v %v", secondaryOne, err)
		return
	}
	klog.Infof("Tunnel created for pod %v at %v \n", secondaryOne, tunnelOne.Local)

	tunnelTwo, err := mongoclient.TunnelToDBPod(k8s.GetRESTConfig(), mg.Namespace, secondaryTwo)
	if err != nil {
		_ = fmt.Errorf("tunnel creation failed for %v %v", secondaryTwo, err)
		return
	}
	klog.Infof("Tunnel created for pod %v at %v \n", secondaryTwo, tunnelTwo.Local)

	so := mongoclient.ConnectToPod(tunnelOne, password)
	defer func() {
		if err := so.Disconnect(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()

	st := mongoclient.ConnectToPod(tunnelTwo, password)
	defer func() {
		if err := st.Disconnect(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()

	Collect(so, Dir+"/"+secondaryOne)
	Collect(st, Dir+"/"+secondaryTwo)

	klog.Infof("Getting stats took %s", time.Since(start))
	klog.Infof("sleep starts. You can run `kubectl cp demo/<doctor-pod>:/app/all-stats /tmp/data` now.")
	time.Sleep(time.Minute * 10)
	//utils.MakeDir(dir)
	//collMap := database.ListCollectionsForAllDatabases(client)
	//for db, collections := range collMap {
	//	if shouldSkip && utils.SkipDB(db) {
	//		continue
	//	}
	//	utils.MakeDir(filepath.Join(dir, db))
	//	dbRef := client.Database(db)
	//	databaseStats(dbRef)
	//	for _, coll := range collections {
	//		if shouldSkip && utils.SkipCollection(coll) {
	//			continue
	//		}
	//		collectionStats(dbRef, coll)
	//	}
	//}
	//admin := client.Database("admin")
	//direct(admin, "serverStatus")
	//direct(admin, "replSetGetStatus")
	//direct(admin, "replSetGetConfig")
	//direct(admin, "currentOp")
	//
	//// error : no such command
	////direct(admin, "getReplicationInfo")
	////direct(admin, "printSecondaryReplicationInfo")
	//indentedData, err := json.MarshalIndent(summed, "", "  ")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//utils.WriteFile(dir, "_", indentedData)
	//klog.Infof("sleep starts. You can run `kubectl cp demo/<doctor-pod>:/app/all-stats /tmp/data` now.")
	//time.Sleep(time.Minute * 10)
}

func Collect(client *mongo.Client, dir string) {
	utils.MakeDir(dir)
	collMap := database.ListCollectionsForAllDatabases(client)
	for db, collections := range collMap {
		if shouldSkip && utils.SkipDB(db) {
			continue
		}
		utils.MakeDir(filepath.Join(dir, db))
		dbRef := client.Database(db)
		databaseStats(dbRef, dir)
		for _, coll := range collections {
			if shouldSkip && utils.SkipCollection(coll) {
				continue
			}
			collectionStats(dbRef, coll, dir)
		}
	}
	admin := client.Database("admin")
	direct(admin, "serverStatus", dir)
	direct(admin, "replSetGetStatus", dir)
	direct(admin, "replSetGetConfig", dir)
	direct(admin, "currentOp", dir)

	// error : no such command
	//direct(admin, "getReplicationInfo")
	//direct(admin, "printSecondaryReplicationInfo")
	indentedData, err := json.MarshalIndent(summed, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	utils.WriteFile(dir, "_", indentedData)
}

func databaseStats(db *mongo.Database, dir string) {
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
