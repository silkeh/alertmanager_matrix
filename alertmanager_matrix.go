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
	room := client.Matrix.NewRoom(mux.Vars(r)["room"])
	if room.ID == "" || room.ID[0] != '!' {
		log.Printf("Invalid room ID: %q", room.ID)
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
	plain, html := client.Formatter.FormatAlerts(data.Alerts, false)
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
	file, err := os.Open(fileName) // nolint:gosec // file inclusion is the point
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
	var addr, iconFile, colorFile string

	config := bot.ClientConfig{Formatter: bot.NewFormatter()}

	flag.StringVar(&addr, "addr", ":4051", "Address to listen on.")
	flag.StringVar(&config.Homeserver, "homeserver", "http://localhost:8008", "Homeserver to connect to.")
	flag.StringVar(&config.UserID, "userID", "", "User ID to connect with.")
	flag.StringVar(&config.Token, "token", "", "Token to connect with.")
	flag.StringVar(&config.Rooms, "rooms", "", "Comma separated list of allowed rooms. All rooms are allowed by default.")
	flag.StringVar(&config.AlertManagerURL, "alertmanager", "http://localhost:9093", "Alertmanager to connect to.")
	flag.StringVar(&config.MessageType, "message-type", "m.notice", "Type of message the bot uses.")
	flag.StringVar(&iconFile, "icon-file", "", "JSON file with icons for message types.")
	flag.StringVar(&colorFile, "color-file", "", "JSON file with colors for message types.")
	flag.Parse()

	// Set variables from the environment
	setStringFromEnv(&addr, "ADDR")
	setStringFromEnv(&config.Homeserver, "HOMESERVER")
	setStringFromEnv(&config.UserID, "USER_ID")
	setStringFromEnv(&config.Token, "TOKEN")
	setStringFromEnv(&config.AlertManagerURL, "ALERTMANAGER")
	setStringFromEnv(&config.Rooms, "ROOMS")

	if iconFile != "" {
		setMapFromJSONFile(&config.Formatter.Icons, iconFile)
	}

	if colorFile != "" {
		setMapFromJSONFile(&config.Formatter.Colors, colorFile)
	}

	if config.UserID == "" || config.Token == "" {
		log.Fatal("Error: user ID or token not supplied")
	}

	log.Printf("Connecting to Matrix homeserver at %s as %s, and to Alertmanager at %s",
		config.Homeserver, config.UserID, config.AlertManagerURL)

	client, err := bot.NewClient(&config)
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
