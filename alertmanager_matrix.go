package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"github.com/silkeh/alertmanager_matrix/alertmanager"
	"github.com/silkeh/alertmanager_matrix/bot"
)

func requestHandler(client *bot.Client, w http.ResponseWriter, r *http.Request) {
	// Get room from request
	roomID := mux.Vars(r)["room"]
	if roomID == "" {
		log.Print("Empty room ID")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	room := client.Matrix.NewRoom(roomID)
	if room.ID[0] != '!' {
		log.Print("Invalid room ID: ", room.ID)
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	// Parse the message
	data := new(alertmanager.Message)
	if err := json.NewDecoder(r.Body).Decode(data); err != nil {
		log.Printf("Error parsing message: %s", err)
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	// Create readable messages for Matrix
	plain, html := bot.FormatAlerts(data.Alerts, false)
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
	file, err := os.Open(fileName) // #nosec
	if err != nil {
		log.Fatalf("Unable to open JSON file %q: %s", fileName, err)
	}

	err = json.NewDecoder(file).Decode(m)
	if err != nil {
		_ = file.Close()

		log.Fatalf("Unable to parse JSON file %q: %s", fileName, err)
	}

	_ = file.Close()
}

func main() {
	var (
		err                              error
		addr                             string // Listen address
		homeserver, userID, token, rooms string // Matrix connection settings
		alertmanagerURL                  string // Alertmanager settings
		iconFile, colorFile, messageType string // Formatting settings
	)

	flag.StringVar(&addr, "addr", ":4051", "Address to listen on.")
	flag.StringVar(&homeserver, "homeserver", "http://localhost:8008", "Homeserver to connect to.")
	flag.StringVar(&userID, "userID", "", "User ID to connect with.")
	flag.StringVar(&token, "token", "", "Token to connect with.")
	flag.StringVar(&rooms, "rooms", "", "Comma separated list of allowed rooms. All rooms are allowed by default.")
	flag.StringVar(&alertmanagerURL, "alertmanager", "http://localhost:9093", "Alertmanager to connect to.")
	flag.StringVar(&iconFile, "icon-file", "", "JSON file with icons for message types.")
	flag.StringVar(&colorFile, "color-file", "", "JSON file with colors for message types.")
	flag.StringVar(&messageType, "message-type", "m.notice", "Type of message the bot uses.")
	flag.Parse()

	// Set variables from the environment
	setStringFromEnv(&addr, "ADDR")
	setStringFromEnv(&homeserver, "HOMESERVER")
	setStringFromEnv(&userID, "USER_ID")
	setStringFromEnv(&token, "TOKEN")
	setStringFromEnv(&alertmanagerURL, "ALERTMANAGER")
	setStringFromEnv(&rooms, "ROOMS")

	// Load mappings from files
	if iconFile != "" {
		setMapFromJSONFile(&bot.AlertIcons, iconFile)
	}

	if colorFile != "" {
		setMapFromJSONFile(&bot.AlertColors, colorFile)
	}

	// Check if user is set
	if userID == "" || token == "" {
		log.Fatal("Error: user ID or token not supplied")
	}

	log.Printf("Connecting to Matrix homeserver at %s as %s, and to Alertmanager at %s",
		homeserver, userID, alertmanagerURL)

	client, err := bot.NewClient(homeserver, userID, token, messageType, rooms, alertmanagerURL)
	if err != nil {
		log.Fatalf("Error connecting to Matrix: %s", err)
	}

	// Start syncing
	go func() {
		log.Fatal(client.Run())
	}()

	// Create the HTTP handler
	handler := func(w http.ResponseWriter, r *http.Request) {
		requestHandler(client, w, r)
	}

	// Create/start HTTP server
	r := mux.NewRouter()
	r.HandleFunc("/{room}", handler).Methods("POST")
	log.Print("Listening on ", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
