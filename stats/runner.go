package stats

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/klog/v2"
	kubedb "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	"kubedb.dev/mongo-doctor/database"
	"kubedb.dev/mongo-doctor/k8s"
	"kubedb.dev/mongo-doctor/mongoclient"
	"kubedb.dev/mongo-doctor/utils"
)

var (
	mg       *kubedb.MongoDB
	password string

	shouldSkip = true
	summed     bson.M

	primaryPod   string
	secondaryOne string
	secondaryTwo string
	err          error
)

func Run(client *mongo.Client) {
	sk := os.Getenv("SKIP")
	if sk == "false" {
		shouldSkip = false
	}

	mg, err = mongoclient.GetMongoDB()
	if err != nil {
		klog.Fatal(err)
	}

	secret, err := mongoclient.GetSecret(mg.Spec.AuthSecret.Name, mg.Namespace)
	if err != nil {
		klog.Fatal(err)
	}
	password = string(secret.Data["password"])

	start := time.Now()
	klog.Infof("STATs starts at %v \n", start)

	if mg.Spec.ReplicaSet != nil {
		runReplicaset(client)
	}
	if mg.Spec.ShardTopology != nil {
		runSharded(mg)
	}
	klog.Infof("Getting stats took %s", time.Since(start))
}

func runReplicaset(client *mongo.Client) {
	hosts, err := database.GetPrimaryAndSecondaries(context.TODO(), client)
	if err != nil {
		_ = fmt.Errorf("error while getting primary and secondaries %v", err)
		return
	}
	primaryPod = hosts[0]
	secondaryOne = hosts[1]
	secondaryTwo = hosts[2]

	klog.Infof("Primary %v and Secondaries found! %v %v \n", primaryPod, secondaryOne, secondaryTwo)

	Collect(client, utils.Dir+"/"+primaryPod, true)

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

	so := mongoclient.ConnectToPod(tunnelOne, mg)
	defer func() {
		if err := so.Disconnect(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()

	st := mongoclient.ConnectToPod(tunnelTwo, mg)
	defer func() {
		if err := st.Disconnect(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()

	Collect(so, utils.Dir+"/"+secondaryOne, true)
	Collect(st, utils.Dir+"/"+secondaryTwo, true)
}

func runSharded(mg *kubedb.MongoDB) {
	getPrimaryAndSecondaries := func(podName string) []string {
		tunn, err := mongoclient.TunnelToDBPod(k8s.GetRESTConfig(), mg.Namespace, podName)
		if err != nil {
			fmt.Errorf("tunnel to pod %s failed: %w", podName, err)
		}
		defer tunn.Close()

		mc := mongoclient.ConnectToPod(tunn, mg)
		defer mc.Disconnect(context.Background())
		list, err := database.GetPrimaryAndSecondaries(context.TODO(), mc)
		if err != nil {
			klog.Errorf("error while getting primary and secondaries %v", err)
		}
		return list
	}

	configServerPods := getPrimaryAndSecondaries(mg.Name + "-configsvr-0")
	shardPods := getPrimaryAndSecondaries(mg.Name + "-shard0-0")

	klog.Infof("configs=%v , shards=%v \n", configServerPods, shardPods)

	baseDir := utils.Dir

	for _, podName := range append(configServerPods, shardPods...) {
		tunn, err := mongoclient.TunnelToDBPod(k8s.GetRESTConfig(), mg.Namespace, podName)
		if err != nil {
			fmt.Errorf("tunnel to pod %s failed: %w", podName, err)
			continue
		}
		defer tunn.Close()

		mc := mongoclient.ConnectToPod(tunn, mg)
		defer mc.Disconnect(context.Background())
		Collect(mc, filepath.Join(baseDir, podName), true)
	}
	for _, podSuffix := range []string{"-mongos-0", "-mongos-1", "-mongos-2"} {
		podName := mg.Name + podSuffix
		tunn, err := mongoclient.TunnelToDBPod(k8s.GetRESTConfig(), mg.Namespace, podName)
		if err != nil {
			fmt.Errorf("tunnel to pod %s failed: %w", podName, err)
			continue
		}
		defer tunn.Close()

		mc := mongoclient.ConnectToPod(tunn, mg)
		defer mc.Disconnect(context.Background())
		Collect(mc, filepath.Join(baseDir, podName), false)
	}
}
