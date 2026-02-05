package mongoclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo/options"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	kubedb "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	"kubedb.dev/mongo-doctor/k8s"
)

func GetMongoClientOptions(mg *kubedb.MongoDB, host string, port int) *options.ClientOptions {
	secret, err := GetSecret(mg.Spec.AuthSecret.Name, mg.Namespace)
	if err != nil {
		klog.Fatal(err)
	}

	if mg.Spec.TLS != nil {
		tlsSecretName := ""
		for _, cert := range mg.Spec.TLS.Certificates {
			if cert.Alias == "client" {
				tlsSecretName = cert.SecretName
			}
		}

		tlsSecret, err := GetSecret(tlsSecretName, mg.Namespace) // adjust name
		if err != nil {
			klog.Fatal(err)
		}

		caPEM := tlsSecret.Data["ca.crt"]
		if len(caPEM) == 0 {
			klog.Fatal("CA certificate not found in secret")
		}

		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(caPEM); !ok {
			klog.Fatal("Failed to append CA cert to pool")
		}

		cert, err := tls.X509KeyPair(
			tlsSecret.Data["tls.crt"],
			tlsSecret.Data["tls.key"], // key is inside the same PEM
		)
		if err != nil {
			klog.Fatal(err)
		}

		tlsConfig := &tls.Config{
			//InsecureSkipVerify: true,
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
			MinVersion:   tls.VersionTLS12, // good default
		}

		cred := options.Credential{
			AuthMechanism: "MONGODB-X509",
			Username:      "CN=root, OU=client, O=kubedb", // must match cert subject
		}

		opts := options.Client().
			ApplyURI(fmt.Sprintf(
				"mongodb://%s:%d/admin?directConnection=true&tls=true",
				host, port,
			)).
			SetTLSConfig(tlsConfig).
			SetAuth(cred).
			SetServerSelectionTimeout(2 * time.Second)

		//uri := fmt.Sprintf(
		//	"mongodb://%s:%s@%s:%v/admin?directConnection=true&tls=true",
		//	string(secret.Data["username"]), string(secret.Data["password"]), host, port)
		//klog.Infof("URI = %s\n", uri)
		//
		//return options.Client().
		//	ApplyURI(uri).
		//	SetTLSConfig(tlsConfig).
		//	SetServerSelectionTimeout(2 * time.Second) // you already had timeout
		return opts
	}

	url := fmt.Sprintf("mongodb://%s:%s@%s:%v/admin?directConnection=true&serverSelectionTimeoutMS=2000&authSource=admin",
		string(secret.Data["username"]), string(secret.Data["password"]), host, port)
	return options.Client().ApplyURI(url)
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
