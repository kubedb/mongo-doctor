package main

import (
	"context"
	"fmt"
	"k8s.io/klog/v2"
	"kubedb.dev/mongo-doctor/mongoclient"
	"kubedb.dev/mongo-doctor/object_count"
	"kubedb.dev/mongo-doctor/stats"
	"log"
	"os"
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

	forURI()
}

func forURI() {
	if uri, exists := os.LookupEnv("MONGODB_URI"); exists {
		client := mongoclient.ConnectFromURI(uri)
		defer func() {
			klog.Infof("disconnecting in defer")
			if err := client.Disconnect(context.Background()); err != nil {
				log.Fatal(err)
			}
		}()
		stats.Collect(client, stats.Dir+"/"+"atlas")
	}
}
