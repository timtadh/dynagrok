package shapes

type Shape Element

type Element interface {
	Location() Point
}

type Point struct {
	X int
	Y int
}

type Window struct {
	Area     *Rectangle
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

func InitWindow(x, y int) *Window {
	rectangle := &Rectangle{Point{0, 0}, x, y}
	return &Window{rectangle, make([]Shape, 0)}
}

/*
func main() {
	w := InitWindow(WindowHeight, WindowWidth)
	circle := &Circle{Origin: Point{w.Height() / 2, w.Width() / 2}}
	circle.SetRadius(5)
	w.AddElement(circle)
	for {
		circle.Move(Point{15, 15})
		if !w.Area.Includes(circle.Location()) {
			break
		}
	}
	w.AddElement(circle)
	w.AddElement(circle)
	w.AddElement(circle)
	fmt.Printf("%v\n", w.Elements)
}
*/

func (r *Rectangle) GetHeight() int {
	return r.Height
}

func (r *Rectangle) GetWidth() int {
	return r.Width
}

func (r *Rectangle) Includes(p Point) bool {
	return p.X < r.Height && p.X > 0 && p.Y < r.Width && p.Y > 0
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

func (w *Window) AddElement(s Element) {
	w.Elements = append(w.Elements, s)
}

func (w *Window) Height() int {
	return w.Area.Height
}

func (w *Window) Width() int {
	return w.Area.Width
}
