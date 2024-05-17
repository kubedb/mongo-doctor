package main

import (
	"context"
	"fmt"
	"k8s.io/klog/v2"
	"kubedb.dev/mongo-doctor/mongoclient"
	"kubedb.dev/mongo-doctor/object_count"
	"kubedb.dev/mongo-doctor/query"
	"kubedb.dev/mongo-doctor/stats"
	"kubedb.dev/mongo-doctor/utils"
	"log"
	"os"
	"time"
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
	object_count.Run(client)
	stats.Run(client)
	query.Run(client, "kubedb_queries")
	forAtlas()
	klog.Infof("sleep starts. You can run `kubectl cp demo/%s:/app/all /tmp/data` now.", os.Getenv("HOSTNAME"))
	time.Sleep(time.Minute * 10)
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
		query.Run(client, "atlas_queries")
	}
}
