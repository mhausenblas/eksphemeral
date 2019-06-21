package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// updateCache updates the cluster spec in the local cache
func updateCache(csstring string) error {
	decoder := json.NewDecoder(strings.NewReader(csstring))
	cs := ClusterSpec{}
	err := decoder.Decode(&cs)
	if err != nil {
		return err
	}
	cscache[cs.ID] = cs
	return nil
}

// invalidateCacheEntry invalidates the cluster spec in the local cache
func invalidateCacheEntry(cID string) {
	if _, ok := cscache[cID]; ok {
		delete(cscache, cID)
	}
}

// lookup tries to look up a cluster spec by ID
func lookup(cID string) (ClusterSpec, error) {
	cs, ok := cscache[cID]
	if ok {
		return cs, nil
	}
	return cs, fmt.Errorf("not cached")
}
