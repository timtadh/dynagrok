package dgtypes

type Type interface {
	Name() string
}

type PrimitiveType struct {
	name string
}

func (p *PrimitiveType) Name() string {
	return p.name
}

type PointerType struct {
	name string
	Elem Type
}

func (p *PointerType) Name() string {
	return p.name
}

type CollectionType struct {
	name string
	Elem Type
	Len  int
	Cap  int
}

func (p *CollectionType) Name() string {
	return p.name
}
