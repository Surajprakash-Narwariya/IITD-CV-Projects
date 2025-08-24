package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

// Cache is a thread-safe in-memory key-value store.
type Cache struct {
	mu   sync.Mutex
	data map[string]string
}

// NewCache creates and returns a new Cache object.
func NewCache() *Cache {
	return &Cache{
		data: make(map[string]string),
	}
}

// Set stores a value for a given key. It is thread-safe.
func (c *Cache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// Get retrieves a value for a given key. It is thread-safe.
func (c *Cache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	val, ok := c.data[key]
	return val, ok
}

// Delete removes a key and its value from the cache. It is thread-safe.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// handleConnection is executed in a new goroutine for each client connection.
// It reads commands from the client, executes them, and writes back the response.
func handleConnection(conn net.Conn, cache *Cache) {
	// Ensure the connection is closed when the function returns.
	defer conn.Close()
	log.Printf("Client connected: %s", conn.RemoteAddr().String())

	// Use a scanner to read line-by-line from the connection.
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		// Get the client's command as a line of text.
		line := scanner.Text()
		// Split the line into parts (command, key, value).
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		// The first part is the command.
		command := strings.ToUpper(parts[0])
		var response string

		// Used a switch to handle different commands.
		switch command {
		case "GET":
			if len(parts) != 2 {
				response = "ERROR: GET command requires exactly one argument (the key)\n"
			} else {
				key := parts[1]
				val, ok := cache.Get(key)
				if !ok {
					response = "NIL\n" // Redis-like response for not found
				} else {
					response = fmt.Sprintf("%s\n", val)
				}
			}
		case "SET":
			if len(parts) < 3 {
				response = "ERROR: SET command requires a key and a value\n"
			} else {
				key := parts[1]
				// Join the rest of the parts to allow values with spaces.
				value := strings.Join(parts[2:], " ")
				cache.Set(key, value)
				response = "OK\n"
			}
		case "DELETE":
			if len(parts) != 2 {
				response = "ERROR: DELETE command requires exactly one argument (the key)\n"
			} else {
				key := parts[1]
				cache.Delete(key)
				response = "OK\n"
			}
		case "QUIT":
			log.Printf("Client disconnected: %s", conn.RemoteAddr().String())
			return // Exit the loop and close the connection.
		default:
			response = fmt.Sprintf("ERROR: Unknown command '%s'\n", command)
		}

		// Write the response back to the client.
		_, err := io.WriteString(conn, response)
		if err != nil {
			log.Printf("Error writing to client %s: %v", conn.RemoteAddr().String(), err)
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from client %s: %v", conn.RemoteAddr().String(), err)
	}
	log.Printf("Connection closed for client: %s", conn.RemoteAddr().String())
}

func main() {
	// Define the port for the cache service.
	port := ":6379"

	// Create a new cache instance.
	cache := NewCache()

	// Start listening for incoming TCP connections.
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to start listener on port %s: %v", port, err)
	}
	// Ensure the listener is closed when main exits.
	defer listener.Close()
	log.Printf("Cache server started, listening on port %s", port)

	// The main loop to accept new client connections.
	for {
		// Block until a new client connects.
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue // Continue to the next iteration to accept another connection.
		}

		// For each new connection, start a new goroutine to handle it concurrently.
		// This is the core of Go's concurrency model for network servers.
		go handleConnection(conn, cache)
	}
}
