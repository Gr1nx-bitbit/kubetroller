package main

import (
	"context"
	"flag"
	"fmt"
	"sync"

	// apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// v1 "k8s.io/client-go/applyconfigurations/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var wg sync.WaitGroup

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

	wg.Add(1)
	go watch(clientset)

	wg.Wait()
}

func intPointer(convert int32) *int32 {
	return &convert
}

func watch(client *kubernetes.Clientset) {
	defer wg.Done()
	watcher, err := client.CoreV1().Pods("default").Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	// result := podWatch.ResultChan()
	// len(result)

	for event := range watcher.ResultChan() {
		fmt.Println(event.Type)
	}
}
