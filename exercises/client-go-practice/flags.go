package main

import (
	"flag"
	"fmt"
	"strings"
)

type Cluster struct {
	clusterName string
	configPath  string
}

func main() {
	var clusterString string
	flag.StringVar(&clusterString, "clusters", "EMPTY", "specify the names of the clusters and their kubeconfig pair in a comma seperated format, e.g. -clusters='name1:config,name2:config' ")
	flag.Parse()

	fmt.Println(getClustersFromFlag(clusterString))
}

func getClustersFromFlag(clusterString string) []Cluster {
	var clusters []Cluster

	for _, clusterPair := range strings.Split(clusterString, ",") {
		pair := strings.Split(clusterPair, ":")
		clusters = append(clusters, Cluster{
			clusterName: pair[0],
			configPath:  pair[1],
		})
	}

	return clusters
}
