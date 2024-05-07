package mongoclient

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	kubedb "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	"kubedb.dev/mongo-doctor/k8s"
	"log"
	"os"
)

func ConnectFromPod() *mongo.Client {
	config := k8s.GetRESTConfig()
	_ = k8s.GetClient(config)
	mg, err := GetMongoDB()
	if err != nil {
		klog.Fatal(err)
	}
	secret, err := GetSecret(mg.Spec.AuthSecret.Name, mg.Namespace)
	if err != nil {
		klog.Fatal(err)
	}

	url := fmt.Sprintf("mongodb://%s:%s@%s.%s.svc:27017/admin?directConnection=true&serverSelectionTimeoutMS=2000&authSource=admin",
		string(secret.Data["username"]), string(secret.Data["password"]), mg.Name, mg.Namespace)
	clientOptions := options.Client().ApplyURI(url)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	klog.Infoln("Connected to MongoDB")
	return client
}

func GetMongoDB() (*kubedb.MongoDB, error) {
	name := os.Getenv("MONGODB_NAME")
	ns := os.Getenv("MONGODB_NAMESPACE")

	var mongodb kubedb.MongoDB
	err := k8s.KBClient.Get(context.TODO(), types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, &mongodb)
	if err != nil {
		return nil, err
	}
	return &mongodb, nil
}

func GetSecret(name, ns string) (*corev1.Secret, error) {
	var authSecret corev1.Secret
	err := k8s.KBClient.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: ns,
	}, &authSecret)
	if err != nil {
		return nil, err
	}
	return &authSecret, nil
}
