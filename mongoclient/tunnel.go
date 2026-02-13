package mongoclient

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/tools/portforward"
	kubedb "kubedb.dev/apimachinery/apis/kubedb/v1"

	"log"
)

func ConnectToPod(tunnel *portforward.Tunnel, mg *kubedb.MongoDB) *mongo.Client {
	client, err := mongo.Connect(context.Background(), GetMongoClientOptions(mg, "localhost", tunnel.Local))
	if err != nil {
		log.Fatal(err)
	}

	if err := client.Ping(context.TODO(), readpref.PrimaryPreferred()); err != nil {
		log.Fatal(err)
	}
	klog.Infof("Connected to secondary %v on port %v \n", tunnel.Name, tunnel.Local)
	return client
}

func TunnelToDBService(config *rest.Config, ns, name string) (*portforward.Tunnel, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	tunnel := portforward.NewTunnel(portforward.TunnelOptions{
		Client:    client.CoreV1().RESTClient(),
		Config:    config,
		Resource:  "services",
		Name:      name,
		Namespace: ns,
		Remote:    27017,
	})

	return tunnel, tunnel.ForwardPort()
}

func TunnelToDBPod(config *rest.Config, ns, podName string) (*portforward.Tunnel, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	tunnel := portforward.NewTunnel(portforward.TunnelOptions{
		Client:    client.CoreV1().RESTClient(),
		Config:    config,
		Resource:  "pods",
		Name:      podName,
		Namespace: ns,
		Remote:    27017,
	})

	return tunnel, tunnel.ForwardPort()
}
