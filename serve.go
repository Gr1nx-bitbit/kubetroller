package main

import (
	"fmt"
	"net/http"
)

func getClusterInfo(writer http.ResponseWriter, req *http.Request) {
	if data, err := getAllClustersData(); err != nil {
		fmt.Printf("Error occured while retrieving data from clsusters! Error: %s\n", err.Error())
		writer.WriteHeader(500)
	} else {
		writer.Write(data)
	}
}

func serve() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", getClusterInfo)

	if err := http.ListenAndServe("localhost:8082", mux); err != nil {
		fmt.Printf("Error while trying to start API!! Error: %s\n", err.Error())
	}
}
