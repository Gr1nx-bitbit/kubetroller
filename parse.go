package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

const (
	ROW     = "<tr>__ROW__</tr>"
	CLUSTER = "<td>__CLUSTER__</td>"
	SERVICE = "<td>__SERVICE__</td>"
	VERSION = "<td style=\"background-color:#__COLOR__\">__VERSION__</td>"
)

func formatData(ctx context.Context, controllers *map[string]*Controller, services *ServiceNames) bool {
	/*
		Ok, so just as our first go, we want to take the names of the clusters
		and make a column for each of them within the html file.
	*/

	if proceed := ctx.Err(); proceed != nil {
		logger := klog.FromContext(ctx)
		logger.Info("Shutting down data formatter")
		return false
	}

	copy := *controllers // So I don't copy the controllers for each for loop (that's also why I pass it in as a pointer)
	clusters := ""
	for cluster := range copy {
		clusters += strings.Replace(CLUSTER, "__CLUSTER__", cluster, 1)
	}

	// ------------------------

	rows := ""
	for service := range services.services {
		rowInner := ""

		rowInner += strings.Replace(SERVICE, "__SERVICE__", service, 1)
		for _, controller := range copy {
			if config, exists := controller.deployments[service]; exists {
				str := strings.Replace(VERSION, "__VERSION__", config.Image, 1)
				str = strings.Replace(str, "__COLOR__", hash(config.Image), 1)
				rowInner += str
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

	return true
}

func hash(convert string) string {
	var total int = 0
	for i := 0; i < len(convert); i++ {
		total += int(byte(convert[i])) * (i + i)
	}

	if total%2 == 0 {
		total += 128
	} else {
		total -= 128
		if total < 0 {
			total *= -1
		}
	}
	hex := fmt.Sprintf("%x", total)
	if length := len(hex); length < 6 {
		for length < 6 {
			hex += "0"
			length++
		}
	} else if length > 6 {
		hex = hex[:5]
	}

	return hex
}
