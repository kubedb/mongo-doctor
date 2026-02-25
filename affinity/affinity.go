package affinity

import (
	"context"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	cu "kmodules.xyz/client-go/client"
	ofstv2 "kmodules.xyz/offshoot-api/api/v2"
	kubedb "kubedb.dev/apimachinery/apis/kubedb/v1"
	"kubedb.dev/mongo-doctor/k8s"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	HardPlacementPolicy = "hard-default"
	ConfigSecretName    = "connection-config"
)

func Run() {
	config := k8s.GetRESTConfig()
	_ = k8s.GetClient(config)
	var mgList kubedb.MongoDBList
	err := k8s.KBClient.List(context.TODO(), &mgList)
	if err != nil {
		log.Fatal(err)
	}
	for _, mg := range mgList.Items {
		if mg.Spec.ShardTopology != nil {
			patchAffinityInOneDatabase(mg)
		}
	}
	for _, podSuffix := range []string{"-mongos-2", "-configsvr-2", "-shard0-1", "-mongos-1", "-configsvr-1", "-shard0-0", "-mongos-0", "-configsvr-0"} {
		klog.Infof("restarting %s pods", podSuffix)
		for _, mg := range mgList.Items {
			if mg.Spec.ShardTopology == nil {
				continue
			}
			var pod corev1.Pod
			err = k8s.KBClient.Get(context.TODO(), types.NamespacedName{
				Namespace: mg.Namespace,
				Name:      mg.Name + podSuffix,
			}, &pod)
			if err != nil {
				klog.Error(err)
				continue
			}
			_ = k8s.KBClient.Delete(context.TODO(), &pod)
		}
		time.Sleep(time.Second * 40)
	}
}

func patchAffinityInOneDatabase(mg kubedb.MongoDB) {
	db := mg.DeepCopy()
	if db.Spec.ShardTopology == nil {
		return
	}

	place := func(spec ofstv2.PodSpec) ofstv2.PodSpec {
		spec.PodPlacementPolicy = &corev1.LocalObjectReference{Name: HardPlacementPolicy}
		return spec
	}
	//val := "-configsvr"
	//if key == kubedb.MongoDBShardLabelKey {
	//	val = "-shard0"
	//} else if key == kubedb.MongoDBMongosLabelKey {
	//	val = "-mongos"
	//}
	//if spec.Affinity == nil {
	//	spec.Affinity = &corev1.Affinity{}
	//	if spec.Affinity.PodAntiAffinity == nil {
	//		spec.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{
	//			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
	//				{
	//					PodAffinityTerm: corev1.PodAffinityTerm{
	//						TopologyKey: "kubernetes.io/hostname",
	//						LabelSelector: &metav1.LabelSelector{
	//							MatchLabels: map[string]string{
	//								key: mg.Name + val,
	//							},
	//						},
	//					},
	//					Weight: 100,
	//				},
	//				{
	//					PodAffinityTerm: corev1.PodAffinityTerm{
	//						TopologyKey: "kubernetes.io/hostname",
	//						LabelSelector: &metav1.LabelSelector{
	//							MatchLabels: map[string]string{
	//								meta.InstanceLabelKey: mg.Name,
	//							},
	//						},
	//					},
	//					Weight: 50,
	//				},
	//			},
	//		}
	//	}
	//}

	db.Spec.ShardTopology.ConfigServer.PodTemplate.Spec = place(db.Spec.ShardTopology.ConfigServer.PodTemplate.Spec)
	db.Spec.ShardTopology.Shard.PodTemplate.Spec = place(db.Spec.ShardTopology.Shard.PodTemplate.Spec)
	db.Spec.ShardTopology.Mongos.PodTemplate.Spec = place(db.Spec.ShardTopology.Mongos.PodTemplate.Spec)

	createConfig := func() error {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ConfigSecretName,
				Namespace: mg.Namespace,
			},
			StringData: map[string]string{
				"mongod.conf": `net:
   maxIncomingConnections: 5000`,
			},
		}
		_, err := cu.CreateOrPatch(context.TODO(), k8s.KBClient, secret, func(obj client.Object, createOp bool) client.Object {
			ret := obj.(*corev1.Secret)
			return ret
		})
		return err
	}

	cnf := func(config *corev1.LocalObjectReference) *corev1.LocalObjectReference {
		if config == nil {
			_ = createConfig()
			config = &corev1.LocalObjectReference{Name: ConfigSecretName}
		}
		return config
	}
	db.Spec.ShardTopology.ConfigServer.ConfigSecret = cnf(db.Spec.ShardTopology.ConfigServer.ConfigSecret)
	db.Spec.ShardTopology.Shard.ConfigSecret = cnf(db.Spec.ShardTopology.Shard.ConfigSecret)
	db.Spec.ShardTopology.Mongos.ConfigSecret = cnf(db.Spec.ShardTopology.Mongos.ConfigSecret)

	_, err := cu.CreateOrPatch(context.TODO(), k8s.KBClient, db, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*kubedb.MongoDB)
		in.Spec = db.Spec
		return in
	})
	if err != nil {
		klog.Error(err)
	}
}
