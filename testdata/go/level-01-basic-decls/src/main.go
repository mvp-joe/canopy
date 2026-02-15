package main

import "fmt"

const MaxRetries = 3

var Debug bool

func Hello() string {
	return "hello"
}

func add(a, b int) int {
	return a + b
}

func main() {
	fmt.Println(Hello())
	result := add(1, 2)
	_ = result
}
