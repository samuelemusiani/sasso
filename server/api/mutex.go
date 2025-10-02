package api

import "sync"

// This file contains the mutex maps that are needed to avoid race conditions
// between different API edpoints.

// Used to serialize access to port forward creation and net deletion
var netMutexes = sync.Map{} // map[uint]*sync.Mutex

func getNetMutex(netID uint) *sync.Mutex {
	mu, _ := netMutexes.LoadOrStore(netID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

// Used to avoid race conditions on VM operations
var vmMutexes = sync.Map{} // map[uint]*sync.Mutex
func getVMMutex(vmID uint) *sync.Mutex {
	mu, _ := vmMutexes.LoadOrStore(vmID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}
