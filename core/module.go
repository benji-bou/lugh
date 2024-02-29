package core

type Modulable interface {
	Load()
}

type Module struct{}

func (m Module) Load() {

}
