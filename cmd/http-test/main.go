package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func all(w http.ResponseWriter, r *http.Request) {
	fmt.Println("all!")
	fmt.Fprintf(w, "all")
}

func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Println("ping!")
	fmt.Fprintf(w, "ping")
}

func pingWithEmail(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	fmt.Printf("ping with Email %s!\n", params["email"])
	fmt.Fprintf(w, "ping with email %s", params["email"])
}

func pingWithNumber(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	fmt.Printf("ping with Number %s!\n", params["number"])
	fmt.Fprintf(w, "ping with number %s", params["number"])
}

type service struct {
	// client dapr.Client
}

func NewService() *service {
	// daprClient, err := dapr.NewClient()
	// if err != nil {
	// 	panic(err)
	// }
	return &service{
		// client: daprClient,
	}
}

func main() {
	fmt.Println("Initializing Ping!")
	// s := NewService()
	// defer s.client.Close()
	// r := mux.NewRouter().StrictSlash(true)
	r := mux.NewRouter()
	r.PathPrefix("/").HandlerFunc(all)

	// r.HandleFunc("/ping", ping)
	// r.HandleFunc("/email/{email}", pingWithEmail)
	// r.HandleFunc("/number/{number}", pingWithNumber)

	log.Fatal(http.ListenAndServe(":8080", r))
}
