package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	// v1 "k8s.io/api/apps/v1"
	// apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func list() {
	kubeconfig := flag.String("kubeconfig", "./config/config", "kubeconfig file")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	list, err := clientset.AppsV1().Deployments("default").List(context.TODO(), metav1.ListOptions{})
	items := list.Items
	for _, item := range items {
		containers := item.Spec.Template.Spec.Containers
		tag := containers[0].Image
		parts := strings.Split(tag, ":")
		fmt.Println(parts[len(parts)-1])
	}
}
