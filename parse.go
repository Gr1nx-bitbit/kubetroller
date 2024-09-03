package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	ROW     = "<tr>__ROW__</tr>"
	CLUSTER = "<td>__CLUSTER__</td>"
	SERVICE = "<td>__SERVICE__</td>"
	VERSION = "<td>__VERSION__</td>"
)

func formatData(controllers *map[string]*Controller, services map[string]interface{}) {
	/*
		Ok, so just as our first go, we want to take the names of the clusters
		and make a column for each of them within the html file.
	*/

	copy := *controllers // So I don't copy the controllers for each for loop
	clusters := ""
	for cluster := range copy {
		clusters += strings.Replace(CLUSTER, "__CLUSTER__", cluster, 1)
	}

	// ------------------------

	rows := ""
	for service := range services {
		rowInner := ""

		rowInner += strings.Replace(SERVICE, "__SERVICE__", service, 1)
		for _, controller := range copy {
			if config, exists := controller.deployments[service]; exists {
				rowInner += strings.Replace(VERSION, "__VERSION__", config.Image, 1)
			} else {
				rowInner += strings.Replace(VERSION, "__VERSION__", "No image found", 1)
			}
		}

		row := strings.Replace(ROW, "__ROW__", rowInner, 1)
		rows += row

	}

	// ------------------------

	bytes, err := os.ReadFile("./templates/allCoallatedTemplate.html")
	if err != nil {
		fmt.Println(err.Error())
	}

	fileContent := string(bytes)
	fileContent = strings.Replace(fileContent, "__DATE__", time.Now().Format("2006-January-02"), 1)
	fileContent = strings.Replace(fileContent, "__CLUSTER_NAMES__", clusters, 1)
	fileContent = strings.Replace(fileContent, "__VERSIONS__", rows, 1)
	os.WriteFile("./out/allCoallated.html", []byte(fileContent), 0644)
}
