package main

import "fmt"

func main() {

	f := test()
	f()


}


func test() func() {

	kk := 5

	return func () {
		fmt.Println(kk)
	}

}
