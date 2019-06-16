package main

import (
	"log"
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir(".")))
	log.Println("EKSPhemeral UI up and running")
	http.ListenAndServe(":8080", nil)
}
