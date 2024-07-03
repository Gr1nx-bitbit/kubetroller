package main

import (
	"context"
	"flag"
	"fmt"

	// apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// v1 "k8s.io/client-go/applyconfigurations/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// change the replicas in the deployment
func main() {
	kubeconfig := flag.String("kubeconfig", "../../config/config", "kubeconfig file")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	deployment, err := clientset.AppsV1().Deployments("default").Get(context.TODO(), "test-deployment", metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	deployment.Spec.Replicas = intPointer(2)

	result, err := clientset.AppsV1().Deployments("default").Update(context.TODO(), deployment, metav1.UpdateOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Println(*result.Spec.Replicas)

}

func intPointer(convert int32) *int32 {
	return &convert
}
