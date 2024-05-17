package mongoclient

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"k8s.io/klog/v2"
	"log"
)

func ConnectFromURI(uri string) *mongo.Client {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	klog.Infoln("Connected to MongoDB with URL")
	return client
}
