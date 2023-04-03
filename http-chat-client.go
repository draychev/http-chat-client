package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var cfg *ChatConfig

// --- Web ChatServerFQDN Endpoints
const (
	endPointSendMessage           = "/send-message"
	endPointGetMessagesForChannel = "/get-messages"
	endPointGetUsersForChannel    = "/get-users"

	// --- HTML Components
	formKeyMessage = "message"

	// --- Default Config Values

	defaultHTTPChatServer      = "chat.server.net"
	defaultHTTPChatPort        = 8080
	defaultWebServerPortNumber = 99

	// --- Environment Variable Keys
	envVarUserNameKey       = "HTTPCHAT_USERNAME"
	envVarConfigFileNameKey = "CONFIG_FILENAME"
)

var (
	envVarUserName       = os.Getenv(envVarUserNameKey)
	envVarConfigFileName = os.Getenv(envVarConfigFileNameKey)
)

type Message struct {
	Username string    `json:"username"`
	Message  string    `json:"message"`
	TimeSent time.Time `json:"timeSent"`
}

type User struct {
	Username string    `json:"username"`
	LastPing time.Time `json:"lastPing"`
}

type Ping struct {
	Username string    `json:"username"`
	TimeSent time.Time `json:"timeSent"`
}

// ChatConfig keeps the config needed to connect to the HTTPChat network
type ChatConfig struct {
	ChatServerFQDN      string `json:"chat-server-fqdn"`
	ChatServerPort      int    `json:"chat-server-port"`
	WebServerPortNumber int    `json:"web-server-port-number"`
}

func readConfig(fileName string) *ChatConfig {
	var config ChatConfig
	// Load the JSON file
	data, err := os.ReadFile(fileName)
	if err != nil {
		log.Fatalf("Failed to read config file [%s]: %s", fileName, err)
	}

	// Parse the JSON into a Config struct
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Println("Failed to parse config file:", err)
		return nil
	}

	if config.Port == 0 {
		config.Port = defaultHTTPChatPort
	}
	if config.ChatServerFQDN == "" {
		config.ChatServerFQDN = defaultHTTPChatServer
	}
	if config.WebServerPortNumber == 0 {
		config.WebServerPortNumber = defaultWebServerPortNumber
	}

	fmt.Printf("Config: %+v\n", config)
	return &config
}

func handlerSendMessage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("Error: %s", err)
		http.Redirect(w, r, "/", 302)
		return
	}
	message := r.Form.Get(formKeyMessage)
	sendMessage(message)
	http.Redirect(w, r, "/", 302)
}

func handlerGetMessagesForChannel(w http.ResponseWriter, r *http.Request) {
	var messages []string
	for idx, msg := range getMessages() {
		messages = append(
			messages,
			fmt.Sprintf("(%d)[%s] %s", idx, msg.Username, msg.Message),
		)
	}
	content := `<!doctype html><html itemscope="" itemtype="http://schema.org/WebPage" lang="en">
	<head><title>smirc: messages</title><meta http-equiv="refresh" content="1"></head>
    <body>` + strings.Join(messages, "<br/>") + `</body></html>`
	_, _ = fmt.Fprintf(w, "%s", content)
}

func handlerGetUsersForChannel(w http.ResponseWriter, r *http.Request) {
	var users []string
	for idx, user := range getActiveUsers() {
		users = append(
			users,
			fmt.Sprintf("(%d)[%s] %s", idx, user.LastPing, user.Username),
		)
	}
	content := `<!doctype html><html itemscope="" itemtype="http://schema.org/WebPage" lang="en">
	<head><title>smirc: users</title><meta http-equiv="refresh" content="5"></head>
    <body><strong>Users:</strong> ` + strings.Join(users, "<br/>") + `</body></html>`
	_, _ = fmt.Fprintf(w, "%s", content)
}

func handlerIndex(w http.ResponseWriter, r *http.Request) {
	content := `<!doctype html><html itemscope="" itemtype="http://schema.org/WebPage" lang="en">
	<head><title>smirc is awesome</title></head><body>
      <iframe marginwidth="0" marginheight="0" width="500" height="500" scrolling="no" frameborder=0 src="` + endPointGetMessagesForChannel + `">
      </iframe>
      <iframe marginwidth="0" marginheight="0" width="500" height="25" scrolling="no" frameborder=0 src="` + endPointGetUsersForChannel + `">
      </iframe>
      <form action="` + endPointSendMessage + `">
        <input type="text" id="` + formKeyMessage + `" name="` + formKeyMessage + `" />
        <input type="submit" value="Send" />
      </form></body></html>`
	_, _ = fmt.Fprintf(w, "%s", content)
}

func sendMessage(message string) {
	log.Printf("Sending message: %s", message)
	httpClient := &http.Client{}

	jsonBytes, _ := json.Marshal(Message{Username: envVarUserName, Message: message})
	url := fmt.Sprintf("http://%s:%d/messages", cfg.ChatServerFQDN, cfg.Port)
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBytes))
	if err != nil {
		log.Fatalf("Failed to post message: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		log.Fatalf("Failed to post message, status code: %d", resp.StatusCode)
	}
}

func getMessages() []Message {
	log.Print("Getting the list of messages...")
	httpClient := &http.Client{}

	// get messages
	url := fmt.Sprintf("http://%s:%d/messages", cfg.ChatServerFQDN, cfg.Port)
	resp, err := httpClient.Get(url)
	if err != nil {
		log.Fatalf("Failed to get messages: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Failed to get messages, status code: %d", resp.StatusCode)
	}
	var messages []Message
	err = json.NewDecoder(resp.Body).Decode(&messages)
	if err != nil {
		log.Fatalf("Failed to decode messages: %v", err)
	}
	return messages
}

func sendPing() {
	log.Print("Sending a PING message")
	httpClient := &http.Client{}

	// ping
	ping := Ping{Username: "Alice"}
	jsonBytes, _ := json.Marshal(ping)
	url := fmt.Sprintf("http://%s:%d/ping", cfg.ChatServerFQDN, cfg.Port)
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBytes))
	if err != nil {
		log.Fatalf("Failed to send a ping: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		log.Fatalf("Failed to send ping, status code: %d", resp.StatusCode)
	}
}

func getActiveUsers() []*User {
	httpClient := &http.Client{}

	// get active users
	url := fmt.Sprintf("http://%s:%d/users", cfg.ChatServerFQDN, cfg.Port)
	resp, err := httpClient.Get(url)
	if err != nil {
		log.Fatalf("Failed to get active users: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Failed to get active users, status code: %d", resp.StatusCode)
	}
	var users []*User
	err = json.NewDecoder(resp.Body).Decode(&users)
	if err != nil {
		log.Fatalf("Failed to decode active users: %v", err)
	}
	return users
}

func main() {
	for _, key := range []string{envVarUserNameKey, envVarConfigFileNameKey} {
		if os.Getenv(key) == "" {
			log.Fatalf("Environment variable %s is required", key)
		}
	}

	cfg = readConfig(envVarConfigFileName)

	http.HandleFunc("/", handlerIndex)
	http.HandleFunc(endPointGetMessagesForChannel, handlerGetMessagesForChannel)
	http.HandleFunc(endPointGetUsersForChannel, handlerGetUsersForChannel)
	http.HandleFunc(endPointSendMessage, handlerSendMessage)

	go func() {
		for {
			sendPing()
			time.Sleep(3 * time.Second)
		}
	}()

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.WebServerPortNumber), nil))
}
