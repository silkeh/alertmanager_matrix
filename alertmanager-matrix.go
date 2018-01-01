package main

import (
	"encoding/json"
	"flag"
	matrix "github.com/matrix-org/gomatrix"
	"log"
	"net/http"
	"os"
)

func handler(w http.ResponseWriter, r *http.Request) {
	data := new(Message)
	if err := json.NewDecoder(r.Body).Decode(data); err != nil {
		log.Printf("Error parsing message: %s", err)
		w.WriteHeader(http.StatusBadRequest)
	}

	plain, html := formatAlerts(data.Alerts)
	log.Printf("Sending message:\n%s", plain)

	if err := sendMessage(plain, html); err != nil {
		log.Printf("Error sending message: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func setStringFromEnv(target *string, env string) {
	if str := os.Getenv(env); str != "" {
		*target = str
	}
}

func main() {
	var addr string
	var homeserver, userID, token string
	var err error

	flag.StringVar(&addr, "addr", ":4051", "Address to listen on.")
	flag.StringVar(&homeserver, "homeserver", "https://matrix.org", "Homeserver to connect to.")
	flag.StringVar(&roomID, "roomID", "", "Room ID to send alerts to.")
	flag.StringVar(&userID, "userID", "", "User ID to connect with.")
	flag.StringVar(&token, "token", "", "Token to connect with")
	flag.Parse()

	setStringFromEnv(&addr, "ADDR")
	setStringFromEnv(&homeserver, "HOMESERVER")
	setStringFromEnv(&roomID, "ROOM_ID")
	setStringFromEnv(&userID, "USER_ID")
	setStringFromEnv(&token, "TOKEN")

	log.Printf("Connecting to Matrix homeserver at %s as %s", homeserver, userID)
	client, err = matrix.NewClient(homeserver, userID, token)
	if err != nil {
		log.Fatalf("Error connecting to Matrix: %s", err)
		os.Exit(1)
	}

	log.Printf("Listening on %s", addr)
	http.HandleFunc("/", handler)
	http.ListenAndServe(addr, nil)
}
