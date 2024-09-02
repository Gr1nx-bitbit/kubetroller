package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	SERVICE = "<td>__SERVICE__</td>"
)

func formatData(controllers *map[string]*Controller) {
	/*
		Ok, so just as our first go, we want to take the names of the clusters
		and make a column for each of them within the html file.
	*/

	services := ""
	for cluster := range *controllers {
		services += strings.Replace(SERVICE, "__SERVICE__", cluster, 1)
	}

	// ------------------------

	bytes, err := os.ReadFile("./templates/allCoallatedTemplate.html")
	if err != nil {
		fmt.Println(err.Error())
	}

	fileContent := string(bytes)
	fileContent = strings.Replace(fileContent, "__DATE__", time.Now().Format("2006-January-02"), 1)
	fileContent = strings.Replace(fileContent, "__SERVICE_NAMES__", services, 1)
	os.WriteFile("./out/allCoallatedTemplate.html", []byte(fileContent), 0644)
}
