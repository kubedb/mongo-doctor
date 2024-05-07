package object_count

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/klog/v2"
	kubedb "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	"kubedb.dev/mongo-doctor/database"
	"kubedb.dev/mongo-doctor/k8s"
	"kubedb.dev/mongo-doctor/mongoclient"
	"log"
	"time"
)

var (
	mg       *kubedb.MongoDB
	password string

	primaryPod   string
	secondaryOne string
	secondaryTwo string
	err          error
)

func Run(client *mongo.Client) {
	mg, err = mongoclient.GetMongoDB()
	if err != nil {
		klog.Fatal(err)
	}

	secret, err := mongoclient.GetSecret(mg.Spec.AuthSecret.Name, mg.Namespace)
	if err != nil {
		klog.Fatal(err)
	}
	password = string(secret.Data["password"])

	klog.Infof("MongoDB found : %v \n", mg.Name)
	hosts, err := database.GetPrimaryAndSecondaries(context.TODO(), client)
	if err != nil {
		_ = fmt.Errorf("error while getting primary and secondaries %v", err)
		return
	}
	primaryPod = hosts[0]
	secondaryOne = hosts[1]
	secondaryTwo = hosts[2]

	klog.Infof("Primary %v and Secondaries found! %v %v \n", primaryPod, secondaryOne, secondaryTwo)

	tunnelOne, err := mongoclient.TunnelToDBPod(k8s.GetRESTConfig(), mg.Namespace, secondaryOne)
	if err != nil {
		_ = fmt.Errorf("tunnel creation failed for %v %v", secondaryOne, err)
		return
	}
	klog.Infof("Tunnel created for pod %v at %v \n", secondaryOne, tunnelOne.Local)

	tunnelTwo, err := mongoclient.TunnelToDBPod(k8s.GetRESTConfig(), mg.Namespace, secondaryTwo)
	if err != nil {
		_ = fmt.Errorf("tunnel creation failed for %v %v", secondaryTwo, err)
		return
	}
	klog.Infof("Tunnel created for pod %v at %v \n", secondaryTwo, tunnelTwo.Local)

	start := time.Now()
	klog.Infof("starts at %v \n", start)
	so := mongoclient.ConnectToPod(tunnelOne, password)
	defer func() {
		if err := so.Disconnect(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()

	st := mongoclient.ConnectToPod(tunnelTwo, password)
	defer func() {
		if err := st.Disconnect(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()

	_, err = CompareObjectsCount(client, so, st)
	if err != nil {
		return
	}
	klog.Infof("Compares took %s", time.Since(start))
}
