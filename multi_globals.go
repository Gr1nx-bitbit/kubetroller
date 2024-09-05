package main

import "sync"

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

	svcNames.services[svcName]++
}

func (svcNames *ServiceNames) decrement(svcName string) {
	svcNames.mutx.Lock()
	defer svcNames.mutx.Unlock()
	value := svcNames.services[svcName]
	if value-1 <= 0 {
		delete(svcNames.services, svcName)
	} else {
		svcNames.services[svcName]--
	}
}
