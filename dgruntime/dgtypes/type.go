package dgtypes

type Type interface {
	Name() string
}

type PrimitiveType struct {
	Tname string
}

func (p *PrimitiveType) Name() string {
	return p.Tname
}

type PointerType struct {
	Tname string
	Elem  Type
}

func (p *PointerType) Name() string {
	return p.Tname
}

type CollectionType struct {
	Tname string
	Elem  Type
	Len   int
	Cap   int
}

func (p *CollectionType) Name() string {
	return p.Tname
}

type StructType struct {
	Tname  string
	Fields []Type
}
