package main

type Shape interface {
	Location() Point
}

type Point struct {
	X int
	Y int
}

type Window struct {
	Area     Rectangle
	Elements []Shape
}

type Rectangle struct {
	Origin Point
	Height int
	Width  int
}

type Circle struct {
	Origin Point
	Radius int
}

const (
	WindowHeight = 400
	WindowWidth  = 600
)

func main() {
	rectangle := Rectangle{Point{0, 0}, WindowHeight, WindowWidth}
	circle := Circle{Origin: Point{rectangle.GetHeight() / 2, rectangle.GetWidth() / 2}}
	circle.SetRadius(5)
	circle.Move(Point{4, 4})
	w := Window{rectangle, make([]Shape, 10)}
	w.Elements = append(w.Elements, circle)
}

func (r *Rectangle) GetHeight() int {
	return r.Height
}

func (r *Rectangle) GetWidth() int {
	return r.Width
}

func (r Rectangle) Location() Point {
	return r.Origin
}

func (c *Circle) SetRadius(v int) {
	c.Radius = v
}

func (c *Circle) Move(p Point) {
	c.Origin.X += p.X
	c.Origin.Y += p.Y
}

func (c Circle) Location() Point {
	return c.Origin
}
