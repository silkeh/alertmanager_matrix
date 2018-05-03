package main

import (
	"encoding/json"
	"flag"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
)

func handler(w http.ResponseWriter, r *http.Request) {
	// Get room from request
	room := client.NewRoom(mux.Vars(r)["room"])
	if room.ID[0] != '!' {
		log.Print("Invalid room ID: ", room.ID)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Parse the message
	data := new(Message)
	if err := json.NewDecoder(r.Body).Decode(data); err != nil {
		log.Printf("Error parsing message: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Create readable messages for Matrix
	plain, html := formatAlerts(data.Alerts)
	log.Printf("Sending message to %s: %s", room.ID, plain)

	if _, err := room.SendHTML(plain, html); err != nil {
		log.Printf("Error sending message: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func setStringFromEnv(target *string, env string) {
	if str := os.Getenv(env); str != "" {
		*target = str
	}
}

func setMapFromJSONFile(m *map[string]string, fileName string) {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal("Unable to open JSON file: ", err)
	}
	err = json.NewDecoder(file).Decode(m)
	if err != nil {
		log.Fatal("Unable to parse JSON file: ", err)
	}
}

func main() {
	var (
		addr                             string // Listen address
		homeserver, userID, token, rooms string // Matrix connection settings
		alertmanager                     string // Alertmanager settings
		iconFile, colorFile, messageType string // Formatting settings
	)

	flag.StringVar(&addr, "addr", ":4051", "Address to listen on.")
	flag.StringVar(&homeserver, "homeserver", "http://localhost:8008", "Homeserver to connect to.")
	flag.StringVar(&userID, "userID", "", "User ID to connect with.")
	flag.StringVar(&token, "token", "", "Token to connect with.")
	flag.StringVar(&rooms, "rooms", "", "Comma separated list of rooms from which commands are allowed. All rooms are allowed by default.")
	flag.StringVar(&alertmanager, "alertmanager", "http://localhost:9093", "Alertmanager to connect to.")
	flag.StringVar(&iconFile, "icon-file", "", "JSON file with icons for message types.")
	flag.StringVar(&colorFile, "color-file", "", "JSON file with colors for message types.")
	flag.StringVar(&messageType, "message-type", "m.notice", "Type of message the bot uses.")
	flag.Parse()

	// Set variables from the environment
	setStringFromEnv(&addr, "ADDR")
	setStringFromEnv(&homeserver, "HOMESERVER")
	setStringFromEnv(&userID, "USER_ID")
	setStringFromEnv(&token, "TOKEN")

	// Load mappings from files
	if iconFile != "" {
		setMapFromJSONFile(&alertIcons, iconFile)
	}
	if colorFile != "" {
		setMapFromJSONFile(&alertColors, colorFile)
	}

	// Check if user is set
	if userID == "" {
		log.Fatal("Error: no user ID")
	}

	// Check if token is set
	if userID == "" {
		log.Fatal("Error: no token")
	}

	log.Printf("Connecting to Matrix homeserver at %s as %s", homeserver, userID)
	err := startMatrixClient(homeserver, userID, token, messageType, rooms)
	if err != nil {
		log.Fatalf("Error connecting to Matrix: %s", err)
	}

	log.Printf("Connecting to Alertmanager at %s", alertmanager)
	err = startAlertmanagerClient(alertmanager)
	if err != nil {
		log.Fatalf("Error connecting to AlertManager: %s", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/{room}", handler).Methods("POST")
	log.Print("Listening on ", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
