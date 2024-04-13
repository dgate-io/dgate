package main

import "fmt"

func main() {
	err := Execute()
	if err != nil {
		fmt.Println(err)
	}
}
