package varret

func Sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, nil
	}
	return a / b, nil
}

func Swap(x, y string) (string, string) {
	return y, x
}
