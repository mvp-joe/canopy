package typeflow

type Shape interface {
	Area() float64
}

type Stringer interface {
	String() string
}
