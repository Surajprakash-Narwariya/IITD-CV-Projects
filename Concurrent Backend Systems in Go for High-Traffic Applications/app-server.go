package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var serverPort string

const (
	rateLimiterAddr = "localhost:7001"
	cacheAddr       = "localhost:6379"
)

func logf(format string, v ...interface{}) {
	log.Printf("[%s] "+format, append([]interface{}{serverPort}, v...)...)
}

// checkRateLimit accepts the client's IP and sends it to the rate limiter.
func checkRateLimit(clientIP string) bool {
	conn, err := net.Dial("tcp", rateLimiterAddr)
	if err != nil {
		logf("Error connecting to rate limiter: %v", err)
		return false
	}
	defer conn.Close()

	// Send the client's IP address to the rate limiter service.
	_, err = conn.Write([]byte(clientIP + "\n"))
	if err != nil {
		logf("Error writing to rate limiter: %v", err)
		return false
	}

	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		logf("Error reading from rate limiter: %v", err)
		return false
	}
	return strings.TrimSpace(response) == "OK"
}

// get the data from the user
func getUserData(userID string) (string, string) {
	cacheKey := fmt.Sprintf("user:%s", userID)
	conn, err := net.Dial("tcp", cacheAddr)
	if err != nil {
		logf("Error connecting to cache: %v", err)
		return fetchFromDatabase(userID), "MISS"
	}
	defer conn.Close()
	getCommand := fmt.Sprintf("GET %s\n", cacheKey)
	_, err = conn.Write([]byte(getCommand))
	if err != nil {
		logf("Error writing to cache: %v", err)
		return fetchFromDatabase(userID), "MISS"
	}
	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		logf("Error reading from cache: %v", err)
		return fetchFromDatabase(userID), "MISS"
	}
	response = strings.TrimSpace(response)
	if response != "NIL" {
		logf("Cache HIT for user %s", userID)
		return response, "HIT"
	}
	logf("Cache MISS for user %s", userID)
	dbData := fetchFromDatabase(userID)
	updateCache(cacheKey, dbData)
	return dbData, "MISS"
}

func fetchFromDatabase(userID string) string {
	time.Sleep(500 * time.Millisecond)
	return fmt.Sprintf("{\"id\": %s, \"name\": \"User %s\", \"data\": \"some-secret-data\"}", userID, userID)
}

func updateCache(key, value string) {
	conn, err := net.Dial("tcp", cacheAddr)
	if err != nil {
		logf("Failed to connect to cache for SET operation: %v", err)
		return
	}
	defer conn.Close()
	setCommand := fmt.Sprintf("SET %s %s\n", key, value)
	_, err = conn.Write([]byte(setCommand))
	if err != nil {
		logf("Failed to write SET command to cache: %v", err)
	}
}

// userHandler extracts the client IP and passes it to the rate limiter.
func userHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the client's IP address from the request.
	// r.RemoteAddr is in the format "ip:port", so we split it.
	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		logf("Could not parse client IP: %v", err)
		return
	}

	// Check the rate limit for this specific client IP.
	if !checkRateLimit(clientIP) {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		logf("Request denied by rate limiter for IP: %s", clientIP)
		return
	}

	if delayParam := r.URL.Query().Get("delay"); delayParam != "" {
		delay, err := strconv.Atoi(delayParam)
		if err == nil {
			logf("Simulating slow request with %dms delay", delay)
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
	}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[2] == "" {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	userID := parts[2]
	data, cacheStatus := getUserData(userID)
	w.Header().Set("X-Cache-Status", cacheStatus)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Handled-By", "app-server-"+serverPort)
	fmt.Fprint(w, data)
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func main() {
	port := flag.String("port", "9001", "Port for the application server to listen on")
	flag.Parse()
	serverPort = *port

	http.HandleFunc("/users/", userHandler)
	http.HandleFunc("/health", healthCheckHandler)

	listenAddr := fmt.Sprintf(":%s", *port)
	logf("Application server starting, listening on http://localhost%s", listenAddr)

	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatalf("Failed to start application server: %v", err)
	}
}
