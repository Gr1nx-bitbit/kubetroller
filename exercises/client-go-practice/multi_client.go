package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// v1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterConfig struct {
	clusterName string
	configPath  string
}

type Cluster struct {
	clusterName string
	configPath  string
	client      kubernetes.Interface
}

func main() {
	var clusterString string
	flag.StringVar(&clusterString, "clusters", "EMPTY", "specify the names of the clusters and their kubeconfig file in a colon-pair comma seperated format, e.g. -clusters='name1:config,name2:config' ")
	flag.Parse()

	// so now that we can get all the kubeconfig files, we have to build each client seperately...
	// idk if trying to build the same client twice will break the program... guess we'll see!
	var clusters []Cluster
	clusterConfigs := getClustersFromFlag(clusterString)
	for index, clusterConfig := range clusterConfigs {
		config, err := clientcmd.BuildConfigFromFlags("", clusterConfig.configPath)
		if err != nil {
			fmt.Println("Something went wrong with cluster config #"+string(index)+"! Error:", err.Error())
			os.Exit(1)
		}

		kclient, err := kubernetes.NewForConfig(config)
		if err != nil {
			fmt.Println("Trouble building client! Error: ", err.Error())
		}

		clusters = append(clusters, Cluster{
			clusterName: clusterConfig.clusterName,
			configPath:  clusterConfig.configPath,
			client:      kclient,
		})
	}

	for index, cluster := range clusters {
		pod, err := cluster.client.CoreV1().Pods("default").Get(context.TODO(), "test", metav1.GetOptions{})
		if err != nil {
			fmt.Println("Error occured while getting pod with client #"+string(index)+"! Error:", err.Error())
		}

		fmt.Printf("Client %s: %s\n", cluster.clusterName, pod.Name)
	}
}

func getClustersFromFlag(clusterString string) []ClusterConfig {
	var clusterConfigs []ClusterConfig

	for _, clusterPair := range strings.Split(clusterString, ",") {
		pair := strings.Split(clusterPair, ":")
		clusterConfigs = append(clusterConfigs, ClusterConfig{
			clusterName: pair[0],
			configPath:  pair[1],
		})
	}

	return clusterConfigs
}
