package closures

func Apply(fn func(int) int, x int) int {
	return fn(x)
}

func Adder(n int) func(int) int {
	return func(x int) int {
		return x + n
	}
}

func main() {
	add5 := Adder(5)
	result := Apply(add5, 10)
	_ = result
}
