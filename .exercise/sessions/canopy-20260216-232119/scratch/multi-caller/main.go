package main

import "fmt"

func helper() string {
    return "hi"
}

func a() { fmt.Println(helper()) }
func b() { fmt.Println(helper()) }
func c() { fmt.Println(helper()) }
func d() { fmt.Println(helper()) }
func e() { fmt.Println(helper()) }

func main() {
    a()
    b()
    c()
    d()
    e()
    fmt.Println(helper())
}
