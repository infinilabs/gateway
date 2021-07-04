package common

import (
	"fmt"
	"sync"

	"infini.sh/framework/core/util"
)

var cacheMap = map[string]map[string]string{}
var lock sync.RWMutex

func GetNodeLevelShuffleKey(cluster, nodeID string) string {

	lock.Lock()
	clusterMap, ok := cacheMap[cluster]
	if !ok {
		clusterMap = map[string]string{}
		cacheMap[cluster] = clusterMap
	}

	key, ok := clusterMap[nodeID]
	if !ok {
		key = fmt.Sprintf("%v-node-%v", cluster, nodeID)
		clusterMap[nodeID] = key
		cacheMap[cluster] = clusterMap
	}
	lock.Unlock()

	return key
}

func GetShardLevelShuffleKey(cluster, index string, shardID int) string {
	lock.Lock()
	clusterMap, ok := cacheMap[cluster]
	if !ok {
		clusterMap = map[string]string{}
		cacheMap[cluster] = clusterMap
	}

	shardStr := util.IntToString(shardID)
	key, ok := clusterMap[shardStr]
	if !ok {
		key = fmt.Sprintf("%v-index-%v-%v", cluster, index, shardID)
		clusterMap[shardStr] = key
		cacheMap[cluster] = clusterMap
	}
	lock.Unlock()
	return key
}
