package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
)

type Node struct {
	NodeId       NodeId
	RoutingTable *RoutingTable
	DHT          DHT
}

func NewNode() *Node {
	key := NewNodeKey()
	return NewNodeWithKey(key)
}

func NewNodeWithId(id NodeId) *Node {
	return &Node{
		NodeId:       id,
		RoutingTable: NewRoutingTable(id),
	}
}

func NewNodeWithPort(port uint16) *Node {
	if port > 65535 {
		panic("Port too big")
	}
	id := NewNodeKey()

	address := fmt.Sprintf("127.0.0.1:%d", port)
	ip, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		panic(err)
	}
	nodeId := NewNodeIdWithIp(id, ip)
	return &Node{
		NodeId:       nodeId,
		RoutingTable: NewRoutingTable(nodeId),
	}
}

func NewNodeWithKey(key Key) *Node {
	nodeId := NewNodeIdWith(key)
	return NewNodeWithId(nodeId)
}

func (n *Node) Start() {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", n.NodeId.IP.String())
	ln, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		panic(err)
		// handle error
	}
	fmt.Printf("start %s\n", n)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				// handle error
				panic(err)
			}
			n.handleConnection(conn)
		}
	}()
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}

func (n *Node) handleConnection(conn net.Conn) {
	defer func() {
		err := conn.Close()
		if err != nil {
			panic(err)
		}
	}()

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	msg := Message{}
	decoder.Decode(&msg)

	if msg.TO != n.NodeId.Key {
		fmt.Println("Ignored. Not targeted node")
		return
	}
	fmt.Printf("%s <<< %s \n", n, msg)

	from := msg.From
	to := msg.TO
	msg.TO = from
	msg.From = to
	switch msg.Type {
	case PING:
		msg.Type = PONG
	case FIND_NODE:
		msg.Nodes = n.RoutingTable.FindClosestBucketById(msg.FindId).nodes
	}

	fmt.Printf("%s >>> %s \n", n, msg)
	encoder.Encode(&msg)
}

// ping a node to find out if is online
func (n *Node) Ping(other *Node) Message {
	fmt.Println(other.NodeId.IP.String())
	conn, err := net.DialTCP("tcp", nil, other.NodeId.IP)
	checkError(err)

	msg := Message{
		Type: PING,
		From: n.NodeId.Key,
		TO:   other.NodeId.Key,
	}
	fmt.Printf("%s >>> %s\n", n, msg)
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	encoder.Encode(msg)
	decoder.Decode(&msg)

	fmt.Printf("%s <<< %s\n", n, msg)
	return msg
}

// call to find a specific node with given id. The recipiend of this call
// looks in it's own routing table and returns a set of contacts that are closeset to
// the NodeId that is being looked up
func (n *Node) FindNode(node NodeId) (*NodeId, error) {
	if n.NodeId == node {
		return nil, errors.New("Can't search for self")
	}
	bucket := n.RoutingTable.FindClosestBucket(&node)
	hasNode, nodeIndex := bucket.Has(node)
	if hasNode {
		get := bucket.Get(nodeIndex)
		return &get, nil
	} else {
		hasNode, _ := bucket.Has(node)
		if hasNode {

		}
		found, err := n.findNodeRemote(node, bucket)
		if err != nil {
			return nil, err
		}
		n.RoutingTable.Add(*found)
		return found, nil
	}
}

// this call tries to find a specific file NodeId to be located. If the receiving
// node finds this NodeId in it's own DHT segment, it will return the corresponding
// URL. If not, the recipient node returns a list of contacts that are closest
// to the file NodeId
func (n *Node) FindValue(value []byte) *FindValueResponse {
	return nil
}

// This call is used to store a key/value pair(fileID,location) in the DHT segment of the recipient node
// Upon each successful RPC, both the sending/receiving node insert/update each other's contact info in their
// own routing table
func (n *Node) Store(value FileID, contact NodeId) {

}

func (n *Node) String() string {
	return fmt.Sprintf("%s", n.NodeId.String())
}

func (n *Node) findNodeRemote(searchedNode NodeId, bucket Bucket) (*NodeId, error) {
	has, i := bucket.Has(n.NodeId)
	if has { // bucket has self
		id := bucket.Get(i)
		return &id, errors.New("Can't search for self")
	}

	for _, nodeId := range bucket.nodes {
		conn, err := net.DialTCP("tcp", nil, nodeId.IP)
		checkError(err)

		msg := Message{
			Type:   FIND_NODE,
			From:   n.NodeId.Key,
			TO:     nodeId.Key,
			FindId: searchedNode.Key,
		}
		fmt.Printf("%s >>> %s\n", n, msg)
		encoder := json.NewEncoder(conn)
		decoder := json.NewDecoder(conn)

		encoder.Encode(msg)
		decoder.Decode(&msg)

		fmt.Printf("%s <<< %s\n", n, msg)

		hasNode, index := msg.Has(searchedNode)
		if hasNode {
			node := msg.Nodes[index]
			return &node, nil
		} else if msg.Nodes !=nil && len(msg.Nodes) != 0 {
			b := NewBucket(msg.Nodes)
			return n.findNodeRemote(searchedNode, b)
		} else {
			continue
		}
	}
	return nil, errors.New("Node not found or not in the network")
}

/**
 *
 */
type FindValueResponse struct {
	ValueFound Segment
	Contacts   []NodeId
}
