package main

import (
	"log"
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir(".")))
	log.Println("EKSPhemeral UI up and running")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
