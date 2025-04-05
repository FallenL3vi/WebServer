package main

import (
	"net/http"
	"fmt"
)

func main() {
	mux := http.NewServeMux()
	server := http.Server{}
	server.Addr =":8080"
	server.Handler = mux

	err := server.ListenAndServe()

	if err != nil {
		fmt.Print("ERROR")
	}
}