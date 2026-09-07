package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/sh", Handler)
	http.HandleFunc("/", redirectHandler)

	fmt.Println(" Server is running")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
