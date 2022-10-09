package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/antonite/ltd-meta-server/server"
)

func main() {
	srv, err := server.New()
	if err != nil {
		panic("failed to create server")
	}

	cert := os.Getenv("cert_path")
	key := os.Getenv("key_path")

	http.HandleFunc("/units", func(w http.ResponseWriter, r *http.Request) {
		srv.HandleGetUnits(w, r)
	})

	http.HandleFunc("/holds", func(w http.ResponseWriter, r *http.Request) {
		srv.HandleGetTopHolds(w, r)
	})

	http.HandleFunc("/versions", func(w http.ResponseWriter, r *http.Request) {
		srv.HandleGetVersions(w, r)
	})

	fmt.Println("Server started   " + time.Now().Format("Mon Jan _2 15:04:05 2006"))

	log.Fatal(http.ListenAndServeTLS(":8081", cert, key, nil))
}
