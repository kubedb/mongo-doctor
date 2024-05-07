package main

import (
	"context"
	"k8s.io/klog/v2"
	"kubedb.dev/mongo-doctor/mongoclient"
	"kubedb.dev/mongo-doctor/object_count"
	"log"
)

func main() {
	client := mongoclient.ConnectFromPod()
	//client := mongoclient.ConnectLocal()
	defer func() {
		klog.Infof("disconnecting in defer")
		if err := client.Disconnect(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()
	object_count.Run(client)
	//stats.Run(client)
}