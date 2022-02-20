package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"

	"github.com/silkeh/alertmanager_matrix/pkg/alertmanager"
	bot2 "github.com/silkeh/alertmanager_matrix/pkg/bot"
)

func requestHandler(client *bot2.Client, w http.ResponseWriter, r *http.Request) {
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

func loadFile(fileName string) string {
	contents, err := os.ReadFile(fileName) // nolint:gosec // contents inclusion is the point
	if err != nil {
		log.Fatalf("Unable to read file %q: %s", fileName, err)
	}

	return string(contents)
}

func mapFromYAMLFile(fileName string) map[string]string {
	file, err := os.Open(fileName) // nolint:gosec // file inclusion is the point
	if err != nil {
		log.Fatalf("Unable to open YAML file %q: %s", fileName, err)
	}

	m := make(map[string]string)

	err = yaml.NewDecoder(file).Decode(m)
	if err != nil {
		_ = file.Close()

		log.Fatalf("Unable to parse YAML file %q: %s", fileName, err)
	}

	_ = file.Close()

	return m
}

func formatter(colorFile, iconFile, htmlTemplateFile, textTemplateFile string) *bot2.Formatter {
	var (
		colors, icons              map[string]string
		htmlTemplate, textTemplate string
	)

	if colorFile != "" {
		colors = mapFromYAMLFile(colorFile)
	}

	if iconFile != "" {
		icons = mapFromYAMLFile(iconFile)
	}

	if htmlTemplateFile != "" {
		htmlTemplate = loadFile(htmlTemplateFile)
	}

	if textTemplateFile != "" {
		textTemplate = loadFile(textTemplateFile)
	}

	return bot2.NewFormatter(textTemplate, htmlTemplate, colors, icons)
}

func main() {
	var addr, iconFile, colorFile, htmlTemplateFile, textTemplateFile string

	config := bot2.ClientConfig{}

	flag.StringVar(&addr, "addr", ":4051", "Address to listen on.")
	flag.StringVar(&config.Homeserver, "homeserver", "http://localhost:8008", "Homeserver to connect to.")
	flag.StringVar(&config.UserID, "userID", "", "User ID to connect with.")
	flag.StringVar(&config.Token, "token", "", "Token to connect with.")
	flag.StringVar(&config.Rooms, "rooms", "", "Comma separated list of allowed rooms. All rooms are allowed by default.")
	flag.StringVar(&config.AlertManagerURL, "alertmanager", "http://localhost:9093", "Alertmanager to connect to.")
	flag.StringVar(&config.MessageType, "message-type", "m.notice", "Type of message the bot uses.")
	flag.StringVar(&iconFile, "icon-file", "", "YAML file with icons for message types.")
	flag.StringVar(&colorFile, "color-file", "", "YAML file with colors for message types.")
	flag.StringVar(&htmlTemplateFile, "html-template", "", "HTML template for alert messages.")
	flag.StringVar(&textTemplateFile, "text-template", "", "Plain-text template for alert messages.")
	flag.Parse()

	// Set variables from the environment
	setStringFromEnv(&addr, "ADDR")
	setStringFromEnv(&config.Homeserver, "HOMESERVER")
	setStringFromEnv(&config.UserID, "USER_ID")
	setStringFromEnv(&config.Token, "TOKEN")
	setStringFromEnv(&config.AlertManagerURL, "ALERTMANAGER")
	setStringFromEnv(&config.Rooms, "ROOMS")

	if config.UserID == "" || config.Token == "" {
		log.Fatal("Error: user ID or token not supplied")
	}

	log.Printf("Connecting to Matrix homeserver at %s as %s, and to Alertmanager at %s",
		config.Homeserver, config.UserID, config.AlertManagerURL)

	client, err := bot2.NewClient(&config, formatter(colorFile, iconFile, htmlTemplateFile, textTemplateFile))
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
