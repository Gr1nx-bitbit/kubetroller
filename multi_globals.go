package main

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/klog/v2"
)

// ok, so when a deployment is deleted, we also need to remove it from here?
// maybe and maybe not... diff clusters can have the same deploy name
type ServiceNames struct {
	services map[string]int
	mutx     sync.Mutex
}

func (svcNames *ServiceNames) checkAndAdd(svcName string) {
	svcNames.mutx.Lock()
	defer svcNames.mutx.Unlock()
	if _, exists := svcNames.services[svcName]; !exists {
		svcNames.services[svcName] = 1
		return
	}

	msg := fmt.Sprintf("num: %d", svcNames.services[svcName]+1)
	klog.Info(msg)
	svcNames.services[svcName]++
}

func (svcNames *ServiceNames) decrement(ctx context.Context, svcName string) {
	svcNames.mutx.Lock()
	defer svcNames.mutx.Unlock()
	value := svcNames.services[svcName]
	logger := klog.FromContext(ctx)
	msg := fmt.Sprintf("value: %d", value)
	logger.Info(msg)
	if value-1 <= 0 {
		delete(svcNames.services, svcName)
	} else {
		svcNames.services[svcName]--
	}
}
