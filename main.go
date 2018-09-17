package streamer

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type channelsResponse struct {
	Channels map[int]string `json:"channels"`
}

func main() {
	// create broadcasting hub
	hub := core.NewHub()

	// add streamers to the hub here..

	// start broadcasting
	hub.Broadcast()

	// serve websockets
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleSocketConnection(w, r, hub)
	})

	// serve endpoints
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/channels", func(w http.ResponseWriter, r *http.Request) {
		handleChannels(w, r, exchanges)
	})

	http.ListenAndServe(":8000", nil)
}

func handleSocketConnection(res http.ResponseWriter, req *http.Request, hub *core.Hub) {
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	// create a listening client
	core.NewClient(conn, hub)
}

// GET index page
func handleIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Go server.")
}

// GET all available channels
func handleChannels(w http.ResponseWriter, r *http.Request, providers []StreamProvider) {
	w.Header().Set("Content-Type", "application/json")
	enableCors(&w)

	// add channels
	chn := make(map[int]string)
	for i := range providers {
		chn[1] = fmt.Sprintf("channel %d", i)
	}
	channels := &channelsResponse{chn}

	// encode json
	json.NewEncoder(w).Encode(channels)
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
