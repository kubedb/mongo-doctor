package mongoclient

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/tools/portforward"
	"log"
)

func ConnectToPod(tunnel *portforward.Tunnel, password string) *mongo.Client {
	url := fmt.Sprintf("mongodb://root:%s@localhost:%v/admin?directConnection=true&serverSelectionTimeoutMS=2000&authSource=admin", password, tunnel.Local)
	clientOptions := options.Client().ApplyURI(url)
	secondaryClient, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	klog.Infof("Connected to secondary %v on port %v \n", tunnel.Name, tunnel.Local)
	return secondaryClient
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
