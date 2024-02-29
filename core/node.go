package core

type Nodes[T any] map[string]Nodable[T]

type Nodable[T any] interface {
	GetName() string
	GetNodes() Nodes[T]
	GetValue() T
}

type BaseNode[T any] struct {
	Name  string                `yaml:"name" json:"name"`
	Nodes map[string]Nodable[T] `yaml:"nodes" json:"nodes"`
}

func (bn BaseNode[T]) GetName() string {
	return bn.Name
}

func (bn BaseNode[T]) GetNodes() Nodes[T] {
	return bn.Nodes
}
