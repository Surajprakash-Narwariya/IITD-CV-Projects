package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// TokenBucket remains the same, representing a single client's rate limit.
type TokenBucket struct {
	capacity          int64
	tokens            int64
	rate              int64
	lastRefillTimestamp time.Time
}

// ClientLimiter holds a map of token buckets, one for each client IP.
type ClientLimiter struct {
	mu      sync.Mutex
	clients map[string]*TokenBucket
}

// NewClientLimiter creates our main rate limiter manager.
func NewClientLimiter() *ClientLimiter {
	return &ClientLimiter{
		clients: make(map[string]*TokenBucket),
	}
}

// Allow checks if a request from a specific clientIP is allowed.
func (cl *ClientLimiter) Allow(clientIP string) bool {
	cl.mu.Lock()
	
	// Check if we have a token bucket for this IP
	bucket, exists := cl.clients[clientIP]
	if !exists {
		// Configure the rate limit for each new client.
		rate := int64(10)     // 10 requests per second
		capacity := int64(20) // with a burst capacity of 20
		bucket = &TokenBucket{
			capacity:          capacity,
			tokens:            capacity,
			rate:              rate,
			lastRefillTimestamp: time.Now(),
		}
		cl.clients[clientIP] = bucket
	}
	
	// Refill the client's specific bucket.
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefillTimestamp)
	tokensToAdd := (elapsed.Nanoseconds() * bucket.rate) / 1e9
	if tokensToAdd > 0 {
		bucket.tokens += tokensToAdd
		if bucket.tokens > bucket.capacity {
			bucket.tokens = bucket.capacity
		}
		bucket.lastRefillTimestamp = now
	}

	cl.mu.Unlock() 

	// Check if the client has tokens.
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// cleanupStaleClients runs in the background to remove old client entries.
func (cl *ClientLimiter) cleanupStaleClients() {
	for {
		// Wait for a minute before cleaning up.
		time.Sleep(1 * time.Minute)

		cl.mu.Lock()
		for ip, bucket := range cl.clients {
			// If a client hasn't been seen for 3 minutes, remove them.
			if time.Since(bucket.lastRefillTimestamp) > 3*time.Minute {
				delete(cl.clients, ip)
				log.Printf("Cleaned up stale rate limiter for IP: %s", ip)
			}
		}
		cl.mu.Unlock()
	}
}

// handleConnection now reads the client's IP to enforce the correct limit.
func handleConnection(conn net.Conn, limiter *ClientLimiter) {
	defer conn.Close()
	
	// Read the client IP from the connection.
	clientIP, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		if err != io.EOF {
			log.Printf("Error reading from client: %v", err)
		}
		return
	}
	clientIP = strings.TrimSpace(clientIP)

	var response string
	if limiter.Allow(clientIP) {
		response = "OK\n"
	} else {
		response = "ERROR\n"
	}

	_, err = io.WriteString(conn, response)
	if err != nil {
		log.Printf("Error writing to client %s: %v", conn.RemoteAddr().String(), err)
	}
}

func main() {
	port := ":7001"
	limiter := NewClientLimiter()

	// Start a background goroutine to clean up old entries.
	go limiter.cleanupStaleClients()

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to start listener on port %s: %v", port, err)
	}
	defer listener.Close()
	log.Printf("Per-client rate limiter started, listening on port %s", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handleConnection(conn, limiter)
	}
}

