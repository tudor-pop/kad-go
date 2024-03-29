package main

import (
	"fmt"
	"log"
)

// number of buckets in the routing table
const Distance_Buckets = 160

type RoutingTable struct {
	currentNode NodeId
	//
	// buckets with index closer to 0 store contacts closer to the current node.
	// The indexing is reversed because is a bit easier to thing that a smaller distance is closer to a node.
	// Nodes that share fewer prefix bits, are further away. Nodes that share many bits, will be closer to 0.
	// the current node is in the first bucket because shared prefix len is 160 - 160 = 0
	// The buckets are reversed because a simple xor with many shared bits, will give many shared 0 prefix values
	// and thus greater indexes e.g. d(00111 ^ 00110) = index(4) and  d(00110 ^ 00110) = index(5) which means
	// the more bits are shared, the further in the list of buckets the node is put. Doing 160 - 4 or 160 - 5
	// will give you the opposite and store closer to 0;
	buckets [Distance_Buckets]Bucket
}

func NewRoutingTable(id NodeId) *RoutingTable {
	rt := &RoutingTable{
		currentNode: id,
		buckets:     [Distance_Buckets]Bucket{},
	}
	rt.Add(id)
	return rt
}

func (rt *RoutingTable) Add(contact NodeId) uint32 {
	index := bucketIndex(rt.currentNode.DistanceTo(&contact))
	rt.buckets[index].Add(contact)
	return index
}

func bucketIndex(prefixLen uint32) uint32 {
	index := Distance_Buckets - prefixLen
	if index == Distance_Buckets {
		index--
	}
	return index
}

func (rt *RoutingTable) IsNodeIdInBucket(id NodeId, index int) bool {
	if index > Distance_Buckets || index < 0 {
		log.Panicf("Invalid index %d\n", index)
		return false
	}
	if index == Distance_Buckets {
		index--
	}
	bucket := rt.buckets[index]
	hasId, _ := bucket.Has(id)
	return hasId
}

func (rt *RoutingTable) Describe() {
	rt.currentNode.Describe()
	for bucket := range rt.buckets {
		fmt.Printf("Bucket %d [", bucket)
		rt.buckets[bucket].Describe()
		fmt.Println("]")
	}
}

func (rt *RoutingTable) FindClosestBucket(id *NodeId) Bucket {
	index := bucketIndex(rt.currentNode.DistanceTo(id))
	for rt.buckets[index].IsEmpty() && index > 0 {
		index--
	}
	return rt.buckets[index]
}
func (rt *RoutingTable) FindClosestBucketById(id Key) Bucket {
	index := bucketIndex(rt.currentNode.Key.SharedPrefixLen(id))
	for rt.buckets[index].IsEmpty() && index > 0 {
		index--
	}
	return rt.buckets[index]
}

func (rt *RoutingTable) LastBucket() Bucket {
	return rt.buckets[len(rt.buckets)-1]
}

func (rt *RoutingTable) LastNotEmptyBucket() Bucket {
	for i := Distance_Buckets - 1; i > 0; i-- {
		if !rt.buckets[i].IsEmpty() {
			return rt.buckets[i]
		}
	}
	return rt.buckets[len(rt.buckets)-1]
}
