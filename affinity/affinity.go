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
	"kmodules.xyz/client-go/meta"
	ofstv1 "kmodules.xyz/offshoot-api/api/v1"
	kubedb "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	"kubedb.dev/mongo-doctor/k8s"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		patchAffinityInOneDatabase(mg)
	}
	for _, mg := range mgList.Items {
		patchAffinityInOneDatabase(mg)
	}
	for _, podSuffix := range []string{"-mongos-2", "-configsvr-2", "-shard0-1", "-mongos-1", "-configsvr-1", "-shard0-0", "-mongos-0", "-configsvr-0"} {
		klog.Infof("restarting %s pods", podSuffix)
		for _, mg := range mgList.Items {
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

	fun := func(spec ofstv1.PodSpec, key string) ofstv1.PodSpec {
		val := "-configsvr"
		if key == kubedb.MongoDBShardLabelKey {
			val = "-shard0"
		} else if key == kubedb.MongoDBMongosLabelKey {
			val = "-mongos"
		}
		if spec.Affinity == nil {
			spec.Affinity = &corev1.Affinity{}
			if spec.Affinity.PodAntiAffinity == nil {
				spec.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							PodAffinityTerm: corev1.PodAffinityTerm{
								TopologyKey: "kubernetes.io/hostname",
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										key: mg.Name + val,
									},
								},
							},
							Weight: 100,
						},
						{
							PodAffinityTerm: corev1.PodAffinityTerm{
								TopologyKey: "kubernetes.io/hostname",
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										meta.InstanceLabelKey: mg.Name,
									},
								},
							},
							Weight: 50,
						},
					},
				}
			}
		}
		return spec
	}

	db.Spec.ShardTopology.ConfigServer.PodTemplate.Spec = fun(db.Spec.ShardTopology.ConfigServer.PodTemplate.Spec, kubedb.MongoDBConfigLabelKey)
	db.Spec.ShardTopology.Shard.PodTemplate.Spec = fun(db.Spec.ShardTopology.Shard.PodTemplate.Spec, kubedb.MongoDBShardLabelKey)
	db.Spec.ShardTopology.Mongos.PodTemplate.Spec = fun(db.Spec.ShardTopology.Mongos.PodTemplate.Spec, kubedb.MongoDBMongosLabelKey)
	_, err := cu.CreateOrPatch(context.TODO(), k8s.KBClient, db, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*kubedb.MongoDB)
		return in
	})
	if err != nil {
		klog.Error(err)
	}
}
