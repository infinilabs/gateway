package common

import "fmt"

func GetNodeLevelShuffleKey(cluster,nodeID string)string  {
	return fmt.Sprintf("%v-node-%v",cluster,nodeID)
}

func GetShardLevelShuffleKey(cluster,index string,shardID int)string  {
	return fmt.Sprintf("%v-index-%v-%v",cluster,index,shardID)
}
