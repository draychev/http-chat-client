package client

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/openservicemesh/osm/pkg/logger"
)

//go:embed style.css
var staticCSS string

var log = logger.New("http-chat-client")

var Config *ChatConfig

// --- Web ChatServerFQDN Endpoints
const (
	EndPointSendMessage = "/send-message"
	EndPointGetMessages = "/get-messages"
	EndPointGetUsers    = "/get-users"

	// --- HTML Components
	formKeyMessage = "message"

	// --- Default Config Values

	defaultHTTPChatServer      = "chat.server.net"
	defaultHTTPChatPort        = 8080
	defaultWebServerPortNumber = 99

	// --- Environment Variable Keys
	EnvVarUserNameKey       = "HTTPCHAT_USERNAME"
	EnvVarConfigFileNameKey = "CONFIG_FILENAME"
)

var (
	EnvVarUserName       = getEnvOrDefault(EnvVarUserNameKey, "change-your-username")
	EnvVarConfigFileName = getEnvOrDefault(EnvVarConfigFileNameKey, "client.Config")
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
		log.Error().Err(err).Msgf("Failed to read config file [%s]: %s", fileName, err)
		return &ChatConfig{
			ChatServerFQDN:      defaultHTTPChatServer,
			ChatServerPort:      defaultHTTPChatPort,
			WebServerPortNumber: defaultWebServerPortNumber,
		}
	}

	// Parse the JSON into a Config struct
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Println("Failed to parse config file:", err)
		return nil
	}

	if config.ChatServerPort == 0 {
		config.ChatServerPort = defaultHTTPChatPort
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

func HandlerSendMessage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Error().Err(err).Msg("Error parsing the web form from request")
		http.Redirect(w, r, "/", 302)
		return
	}
	message := r.Form.Get(formKeyMessage)
	sendMessage(message)
	http.Redirect(w, r, "/", 302)
}

func HandlerGetMessages(w http.ResponseWriter, r *http.Request) {
	var messages []string
	for idx, msg := range getMessages() {
		messages = append(
			messages,
			fmt.Sprintf("(%d)[%s] %s", idx, msg.Username, msg.Message),
		)
	}
	content := `<!doctype html><html itemscope="" itemtype="http://schema.org/WebPage" lang="en">
	<head><title>http-chat-client: messages</title><meta http-equiv="refresh" content="1">` + getCSS() + `</head>
    <body><div><strong>Chat Messages:</strong><br/>` + strings.Join(messages, "<br/>") + `</div></body></html>`
	_, _ = fmt.Fprintf(w, "%s", content)
}

func HandlerGetUsers(w http.ResponseWriter, r *http.Request) {
	var users []string
	for idx, user := range getActiveUsers() {
		users = append(
			users,
			fmt.Sprintf("(%d) %s", idx, user.Username),
		)
	}
	content := `<!doctype html><html itemscope="" itemtype="http://schema.org/WebPage" lang="en">
	<head><title>http-chat-client: users</title><meta http-equiv="refresh" content="5">` + getCSS() + `</head>
    <body><div><strong>Users:</strong><br/> ` + strings.Join(users, "<br/>") + `</div></body></html>`
	_, _ = fmt.Fprintf(w, "%s", content)
}

func getCSS() string {
	return fmt.Sprintf(`<style type="text/css">%s</style>`, staticCSS)
}

func HandlerIndex(w http.ResponseWriter, r *http.Request) {
	content := `<!doctype html><html itemscope="" itemtype="http://schema.org/WebPage" lang="en">
	<head><title>http-chat-client is awesome</title><style></style>` + getCSS() + `</head><body>
      <table><tr><td>
      <iframe marginwidth="0" marginheight="0" width="480" height="640" scrolling="yes" frameborder=0 src="` + EndPointGetMessages + `">
      </iframe>
      </td><td>
      <iframe marginwidth="0" marginheight="0" width="480" height="640" scrolling="yes" frameborder=0 src="` + EndPointGetUsers + `">
      </iframe>
      </td></tr></table>
      <form method="post" action="` + EndPointSendMessage + `">
        <input type="text" id="` + formKeyMessage + `" name="` + formKeyMessage + `" />
        <input type="submit" value="Send" />
      </form></body></html>`
	_, _ = fmt.Fprintf(w, "%s", content)
}

func sendMessage(message string) {
	log.Info().Msgf("Sending message: %s", message)
	httpClient := &http.Client{}
	jsonBytes, _ := json.Marshal(Message{Username: EnvVarUserName, Message: message})
	url := fmt.Sprintf("http://%s:%d/messages", Config.ChatServerFQDN, Config.ChatServerPort)
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBytes))
	if err != nil {
		log.Error().Err(err).Msgf("Failed to POST message to %s: %s", url, jsonBytes)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		log.Error().Msgf("Failed to POST message to %s, status code: %d", url, resp.StatusCode)
		return
	}
}

func getMessages() []Message {
	log.Info().Msg("Getting the list of messages...")
	httpClient := &http.Client{}
	var messages []Message
	// get messages
	url := fmt.Sprintf("http://%s:%d/messages", Config.ChatServerFQDN, Config.ChatServerPort)
	resp, err := httpClient.Get(url)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get messages")
		return messages
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("Failed to get messages, status code: %d", resp.StatusCode)
		return messages
	}
	err = json.NewDecoder(resp.Body).Decode(&messages)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to decode messages: %+v", messages)
		return messages
	}
	return messages
}

func sendPing() {
	httpClient := &http.Client{}
	ping := Ping{Username: EnvVarUserName}
	jsonBytes, _ := json.Marshal(ping)
	url := fmt.Sprintf("http://%s:%d/ping", Config.ChatServerFQDN, Config.ChatServerPort)
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBytes))
	if err != nil {
		log.Error().Err(err).Msgf("Failed to send a ping to : %s", url)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		log.Error().Msgf("Failed to send ping to %s, status code: %d", url, resp.StatusCode)
		return
	}
	log.Info().Msgf("Sent a PING: %s", ping)
}

func getActiveUsers() []*User {
	httpClient := &http.Client{}
	var users []*User
	url := fmt.Sprintf("http://%s:%d/users", Config.ChatServerFQDN, Config.ChatServerPort)
	resp, err := httpClient.Get(url)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get active users from %s", url)
		return users
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("Failed to get active users from %s, status code: %d", url, resp.StatusCode)
		return users
	}
	err = json.NewDecoder(resp.Body).Decode(&users)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to decode active users from %s: %v", url, err)
		return users
	}
	return users
}

func NewChatClient(quit chan interface{}, ready chan interface{}) {
	for _, key := range []string{EnvVarUserNameKey, EnvVarConfigFileNameKey} {
		if os.Getenv(key) == "" {
			log.Fatal().Msgf("Environment variable %s is required", key)
		}
	}

	Config = readConfig(EnvVarConfigFileName)

	http.HandleFunc("/", HandlerIndex)
	http.HandleFunc(EndPointGetMessages, HandlerGetMessages)
	http.HandleFunc(EndPointGetUsers, HandlerGetUsers)
	http.HandleFunc(EndPointSendMessage, HandlerSendMessage)

	ticker := time.NewTicker(3000 * time.Millisecond)
	done := make(chan bool)

	defer func() {
		ticker.Stop()
		done <- true
	}()

	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-ticker.C:
				log.Info().Msgf("Trying to send a PING %+v...", time.Now())
				sendPing()
			}
		}
	}()
	ready <- true
	<-quit
}

func getEnvOrDefault(key, defaultValue string) string {
	if EnvVarValue := os.Getenv(key); EnvVarValue != "" {
		return EnvVarValue
	}
	return defaultValue
}
