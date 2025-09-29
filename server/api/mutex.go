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
