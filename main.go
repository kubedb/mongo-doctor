package main

import (
	"context"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"k8s.io/klog/v2"
	kubedb "kubedb.dev/apimachinery/apis/kubedb/v1"
	"kubedb.dev/mongo-doctor/affinity"
	"kubedb.dev/mongo-doctor/k8s"
	"kubedb.dev/mongo-doctor/mongoclient"
	"kubedb.dev/mongo-doctor/stats"
	"kubedb.dev/mongo-doctor/utils"
)

func main() {
	klog.Infof("Starting MongoDB Doctor")

	affinity.Run()
	time.Sleep(time.Second * 40)
	klog.Infof("Affinity done. Starting stat...")
	config := k8s.GetRESTConfig()
	_ = k8s.GetClient(config)
	var mgList kubedb.MongoDBList
	err := k8s.KBClient.List(context.TODO(), &mgList)
	if err != nil {
		log.Fatal(err)
	}
	for _, mg := range mgList.Items {
		host := mongoclient.DefaultHost(&mg)
		client, err := mongo.Connect(context.Background(), mongoclient.GetMongoClientOptions(&mg, host, 27017))
		if err != nil {
			log.Fatal(err)
		}

		if err := client.Ping(context.TODO(), readpref.PrimaryPreferred()); err != nil {
			log.Fatal(err)
		}
		klog.Infof("Starting stat collection for MongoDB %s/%s", mg.Namespace, mg.Name)
		stats.Run(client)
	}

	//client := mongoclient.ConnectFromPod()
	//defer func() {
	//	klog.Infof("disconnecting in defer")
	//	if err := client.Disconnect(context.Background()); err != nil {
	//		log.Fatal(err)
	//	}
	//}()

	//uri, exists := os.LookupEnv("MONGODB_URI")
	//if !exists {
	//	log.Fatal("MONGODB_URI env not set")
	//}
	//atlasClient := mongoclient.ConnectFromURI(uri)
	//defer func() {
	//	klog.Infof("disconnecting in defer")
	//	if err := atlasClient.Disconnect(context.Background()); err != nil {
	//		log.Fatal(err)
	//	}
	//}()
	//diffchecker.Run(client, atlasClient)
	//fun(client)
	//object_count.Run(client)
	//stats.Run(client)
	//query.Run(client, "kubedb_queries")
	//forAtlas()
	klog.Infof("sleep starts. You can run `kubectl cp demo/%s:/all /tmp/data` now.", os.Getenv("HOSTNAME"))
	time.Sleep(time.Minute * 10)
}

func fun(client *mongo.Client) {
	db := client.Database("aa")
	stat := collectionStats(db, "one")

	klog.Infof("nIndexes: %v", reflect.TypeOf(stat["nindexes"]).Name())
	klog.Infof("indexSizes: %v %v", reflect.TypeOf(stat["indexSizes"]).Name(), stat["indexSizes"])
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

func forAtlas() {
	if uri, exists := os.LookupEnv("MONGODB_URI"); exists {
		client := mongoclient.ConnectFromURI(uri)
		defer func() {
			klog.Infof("disconnecting in defer")
			if err := client.Disconnect(context.Background()); err != nil {
				log.Fatal(err)
			}
		}()
		stats.Collect(client, utils.Dir+"/"+"atlas", true)
		//query.Run(client, "atlas_queries")
	}
}
