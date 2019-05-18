package main

import (
	"airviz/datasrc"
	"airviz/latest"
	. "airviz/server"
	"flag"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":4000", "http service address")

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

func main() {
	flag.Parse()
	hub := NewHub()
	go hub.Run()

	blocks := latest.NewDag(300)
	mockBlocksSrc := datasrc.Mocksrc{Dag: blocks}
	go mockBlocksSrc.Start()

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, blocks, w, r)
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
