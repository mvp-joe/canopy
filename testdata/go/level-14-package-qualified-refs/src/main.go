package main

import "mathutil"

func Calculate() int {
	sum := mathutil.Add(1, 2)
	product := mathutil.Multiply(3, 4)
	return sum + product
}
