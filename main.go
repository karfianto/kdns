// main.go

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

// Initialize a logger
var logger = logrus.New()

func init() {
	// Use JSON formatter for log output
	logger.Formatter = &logrus.JSONFormatter{}
}

type DNSConfig struct {
	ServerAddr   string       `json:"ServerAddr"`
	ExternalAddr string       `json:"ExternalAddr"`
	Zones        []ZoneConfig `json:"Zones"`
}

type ZoneConfig struct {
	Zone    string                  `json:"Zone"`
	Records map[string]RecordConfig `json:"Records"`
}

type RecordConfig struct {
	Type string `json:"Type"`
	Data string `json:"Data"`
}

var currentConfig DNSConfig

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan string)

func main() {

	// Load environment variables from the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Retrieve environment variables
	bindURL := os.Getenv("HTTP_BIND_URL")
	port := os.Getenv("HTTP_PORT")
	debug := os.Getenv("HTTP_DEBUG")

	// Print the environment variables for testing
	fmt.Printf("BIND_URL: %s\n", bindURL)
	fmt.Printf("PORT: %s\n", port)
	fmt.Printf("DEBUG: %s\n", debug)

	// Load DNS configuration
	if err := LoadConfigFromFile("dnsconfig.json", &currentConfig); err != nil {
		fmt.Printf("Error loading DNS config: %s\n", err)
		return
	}

	// Initialize the Gin router for the API
	router := InitRouter()

	// Start the DNS server
	go StartDNSServer(debug)
	go handleMessages()

	// Start the API server on port 8080
	err = router.Run(":" + port)
	if err != nil {
		fmt.Printf("Error starting API server: %s\n", err)
	}
}

func LoadConfigFromFile(filename string, config *DNSConfig) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, config)
}

func SaveConfigToFile(filename string, config DNSConfig) error {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

func InitRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/api/config", getConfig)
	r.POST("/api/config", updateConfig)
	// Start WebSocket server
	r.GET("/ws", handleWebSocket)
	r.GET("/", handleHTTP)

	return r
}

func getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, currentConfig)
}

func updateConfig(c *gin.Context) {
	var newConfig DNSConfig
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the new configuration (add your validation logic here)

	currentConfig = newConfig

	if err := SaveConfigToFile("dnsconfig.json", currentConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuration updated successfully"})
}

func StartDNSServer(d string) {
	server := &dns.Server{Addr: currentConfig.ServerAddr, Net: "udp"}

	server.Handler = dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		msg := new(dns.Msg)
		msg.SetReply(r)

		// Process each DNS question in the query
		for _, q := range r.Question {
			zone := getMatchingZone(q.Name)

			if zone != nil {
				if record, ok := zone.Records[strings.ToLower(q.Name)]; ok {
					// Check if the requested record type matches the configured type
					if q.Qtype == dns.StringToType[record.Type] {
						// Respond with the configured DNS record data
						switch q.Qtype {
						case dns.TypeA:
							answer, err := dns.NewRR(fmt.Sprintf("%s IN %s %s", q.Name, record.Type, record.Data))
							if err != nil {
								fmt.Printf("Error creating DNS record: %s\n", err)
							} else {
								msg.Answer = append(msg.Answer, answer)
							}
						case dns.TypeMX:
							// Example of creating an MX record (Mail Exchanger record)
							// Replace with your own logic
							answer, err := dns.NewRR(fmt.Sprintf("%s IN %s %s", q.Name, record.Type, record.Data))
							if err != nil {
								fmt.Printf("Error creating DNS record: %s\n", err)
							} else {
								msg.Answer = append(msg.Answer, answer)
							}
						case dns.TypeTXT:
							// Example of creating a TXT record
							// Replace with your own logic
							answer, err := dns.NewRR(fmt.Sprintf("%s IN %s \"%s\"", q.Name, record.Type, record.Data))
							if err != nil {
								fmt.Printf("Error creating DNS record: %s\n", err)
							} else {
								msg.Answer = append(msg.Answer, answer)
							}
						default:
							fmt.Printf("Unsupported record type: %s\n", dns.TypeToString[q.Qtype])
						}
					}
				}
			} else {
				// If not found in configured zones, forward the query to the external resolver
				extResponse, err := dns.Exchange(r, currentConfig.ExternalAddr)
				if err == nil {
					msg.Answer = extResponse.Answer
				}
			}
		}

		logDNSRequest(w, r, d)

		// Send the DNS response
		w.WriteMsg(msg)
	})

	fmt.Printf("DNS server listening on %s\n", currentConfig.ServerAddr)

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			fmt.Printf("Error starting DNS server: %s\n", err)
		}
	}()

	// Periodically check for config changes and reload
	go func() {
		for {
			time.Sleep(5 * time.Second)

			var newConfig DNSConfig
			if LoadConfigFromFile("dnsconfig.json", &newConfig) != nil {
				if !configEqual(currentConfig, newConfig) {
					fmt.Println("Reloading DNS configuration...")
					currentConfig = newConfig

					if err := SaveConfigToFile("dnsconfig.json", currentConfig); err != nil {
						fmt.Printf("Error saving updated config: %s\n", err)
					}
				}
			}
		}
	}()

	select {}
}

func configEqual(config1, config2 DNSConfig) bool {
	if config1.ServerAddr != config2.ServerAddr || config1.ExternalAddr != config2.ExternalAddr {
		return false
	}

	if len(config1.Zones) != len(config2.Zones) {
		return false
	}

	for i, zone1 := range config1.Zones {
		zone2 := config2.Zones[i]
		if zone1.Zone != zone2.Zone || !mapEqual(zone1.Records, zone2.Records) {
			return false
		}
	}

	return true
}

func mapEqual(m1, m2 map[string]RecordConfig) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v1 := range m1 {
		v2, ok := m2[k]
		if !ok || v1.Type != v2.Type || v1.Data != v2.Data {
			return false
		}
	}
	return true
}

func getMatchingZone(name string) *ZoneConfig {
	for _, zone := range currentConfig.Zones {
		if strings.HasSuffix(strings.ToLower(name), zone.Zone) {
			return &zone
		}
	}
	return nil
}

func logDNSRequest(w dns.ResponseWriter, r *dns.Msg, d string) {
	// Extract relevant information from the DNS request
	srcIP, _, _ := net.SplitHostPort(w.RemoteAddr().String())
	for _, q := range r.Question {
		logMessage := fmt.Sprintf("Received DNS request from %s for %s (Type: %s)", srcIP, q.Name, dns.TypeToString[q.Qtype])
		// Send the log message to WebSocket clients
		broadcast <- logMessage

		// Log the message to the server logs
		// logger.Info(logMessage)

		if d == "true" {
			logger.WithFields(logrus.Fields{
				"SourceIP":  srcIP,
				"QueryName": q.Name,
				"QueryType": dns.TypeToString[q.Qtype],
			}).Info("Received DNS request")

		}
	}
}

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	clients[conn] = true

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			delete(clients, conn)
			return
		}

		broadcast <- string(msg)
	}
}

func handleMessages() {
	for {
		msg := <-broadcast

		// Send the message to all connected clients
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				log.Println(err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
func handleHTTP(c *gin.Context) {
	// Serve the HTML file
	c.File("index.html")
}
