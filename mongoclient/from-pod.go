package mongoclient

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"k8s.io/klog/v2"
	kubedb "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	"kubedb.dev/mongo-doctor/k8s"
)

func ConnectFromPod() *mongo.Client {
	config := k8s.GetRESTConfig()
	_ = k8s.GetClient(config)
	mg, err := GetMongoDB()
	if err != nil {
		klog.Fatal(err)
	}

	host := DefaultHost(mg)
	client, err := mongo.Connect(context.Background(), GetMongoClientOptions(mg, host, 27017))
	if err != nil {
		log.Fatal(err)
	}

	if err := client.Ping(context.TODO(), readpref.PrimaryPreferred()); err != nil {
		log.Fatal(err)
	}
	klog.Infoln("Connected to MongoDB")
	return client
}

func DefaultHost(mg *kubedb.MongoDB) string {
	return fmt.Sprintf("%s.%s.svc", mg.Name, mg.Namespace)
}
