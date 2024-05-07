package k8s

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubedbscheme "kubedb.dev/apimachinery/client/clientset/versioned/scheme"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	KBClient client.Client
	scm      = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scm))
	utilruntime.Must(kubedbscheme.AddToScheme(scm))
}

func GetRESTConfig() *rest.Config {
	kubeConfig := os.Getenv("KUBECONFIG")
	var (
		config *rest.Config
		err    error
	)

	if kubeConfig == "" {
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			panic(err.Error())
		}
	}
	return config
}

func GetClient(config *rest.Config) client.Client {
	kc, err := client.New(config, client.Options{
		Scheme: scm,
		Mapper: nil,
	})
	if err != nil {
		panic(err)
	}
	KBClient = kc
	return kc
}
