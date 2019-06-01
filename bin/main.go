package main

import (
	"airviz/core"
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

	// create a hub for clients to connect to
	hub := NewHub()

	// create aa server state to maintain the collection of dags in, and route events through
	state := NewServerState(hub)
	state.SetDag(core.TopicBlocks, latest.NewDag(300))

	// route events to the dags
	go state.PipeEvents()

	// get event channel, this is where all sources send their data to get it into the dag.
	evs := state.GetEventCh()

	// attach data sources
	mockBlocksSrc := datasrc.Mocksrc{}
	go mockBlocksSrc.Start(evs)

	// enable clients to connect
	go hub.Run()

	// route to http handlers. There's a home page and a websocket entry.
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", state.ServeWs)

	// accept connections
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
