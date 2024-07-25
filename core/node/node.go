package core

import "golang.org/x/exp/maps"

type NodeIndex map[string]string

type Nodes[T any] map[string]Nodable[T]

type IndexableNode[T any] interface {
	BuildIndex()
	Query(query T)
}

type Nodable[T any] interface {
	GetName() string
	GetChilds() Nodes[T]
	GetValue() T
}

type MutableNode[T any] interface {
	Nodable[T]
	AddNode(node ...Nodable[T])
	DeleteNode(node ...string)
}

type Node[T any] struct {
	Name  string                `yaml:"name" json:"name"`
	Nodes map[string]Nodable[T] `yaml:"nodes" json:"nodes"`
	Index NodeIndex
}

func (bn Node[T]) GetName() string {
	return bn.Name
}

func (bn Node[T]) GetChilds() Nodes[T] {
	return bn.Nodes
}

func (bn *Node[T]) AddNode(node ...Nodable[T]) {
	for _, n := range node {
		bn.Nodes[n.GetName()] = n
	}
}

func (bn *Node[T]) DeleteNode(node ...string) {
	for _, n := range node {
		delete(bn.Nodes, n)
	}
}

func Walk[T any](root Nodable[T], nextNodesCb func(parent, node Nodable[T])) {
	nextNodes := []Nodable[T]{root}
	var parentNode Nodable[T] = nil
	for len(nextNodes) > 0 {
		parentNode = nextNodes[0]
		sliceChilds := maps.Values(parentNode.GetChilds())
		nextNodes = append(nextNodes[1:], sliceChilds...)
		for _, c := range sliceChilds {
			nextNodesCb(parentNode, c)
		}
	}
}
