package common

import (
	"fmt"
)

//var cacheMap = map[string]map[string]string{}
//var lock sync.RWMutex

func GetNodeLevelShuffleKey(cluster, nodeID string) string {
	return fmt.Sprintf("%v-node-%v", cluster, nodeID)

	//clusterMap, ok := cacheMap[cluster]
	//if !ok {
	//	lock.Lock()
	//	clusterMap = map[string]string{}
	//	cacheMap[cluster] = clusterMap
	//	lock.Unlock()
	//}
	//
	//key, ok := clusterMap[nodeID]
	//if !ok {
	//	lock.Lock()
	//	key = fmt.Sprintf("%v-node-%v", cluster, nodeID)
	//	clusterMap[nodeID] = key
	//	cacheMap[cluster] = clusterMap
	//	lock.Unlock()
	//}
	//
	//return key
}

func GetShardLevelShuffleKey(cluster, index string, shardID int) string {

	return fmt.Sprintf("%v-index-%v-%v", cluster, index, shardID)

	//clusterMap, ok := cacheMap[cluster]
	//if !ok {
	//	lock.Lock()
	//	clusterMap = map[string]string{}
	//	cacheMap[cluster] = clusterMap
	//	lock.Unlock()
	//}
	//
	//shardStr := util.IntToString(shardID)
	//key, ok := clusterMap[shardStr]
	//if !ok {
	//	lock.Lock()
	//	key = fmt.Sprintf("%v-index-%v-%v", cluster, index, shardID)
	//	clusterMap[shardStr] = key
	//	cacheMap[cluster] = clusterMap
	//	lock.Unlock()
	//}
	//
	//return key
}
