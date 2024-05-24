package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/klog/v2"
	"kubedb.dev/mongo-doctor/diffchecker"
	"kubedb.dev/mongo-doctor/mongoclient"
	"kubedb.dev/mongo-doctor/stats"
	"kubedb.dev/mongo-doctor/utils"
	"log"
	"os"
	"reflect"
	"strings"
)

func main() {
	fmt.Println("Starting MongoDB Doctor")

	client := mongoclient.ConnectFromPod()
	defer func() {
		klog.Infof("disconnecting in defer")
		if err := client.Disconnect(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()

	uri, exists := os.LookupEnv("MONGODB_URI")
	if !exists {
		log.Fatal("MONGODB_URI env not set")
	}
	atlasClient := mongoclient.ConnectFromURI(uri)
	defer func() {
		klog.Infof("disconnecting in defer")
		if err := atlasClient.Disconnect(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()
	diffchecker.Run(client, atlasClient)
	//fun(client)
	//object_count.Run(client)
	//stats.Run(client)
	//query.Run(client, "kubedb_queries")
	//forAtlas()
	//klog.Infof("sleep starts. You can run `kubectl cp demo/%s:/app/all /tmp/data` now.", os.Getenv("HOSTNAME"))
	//time.Sleep(time.Minute * 10)
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
		stats.Collect(client, utils.Dir+"/"+"atlas")
		//query.Run(client, "atlas_queries")
	}
}
