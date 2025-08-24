package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// server represents a backend application server.
// It includes a health status.
type server struct {
	URL   *url.URL
	Proxy *httputil.ReverseProxy
	mu    sync.RWMutex 
	alive bool
}

// SetAlive sets the health status of the server.
func (s *server) SetAlive(alive bool) {
	s.mu.Lock()
	s.alive = alive
	s.mu.Unlock()
}

// IsAlive checks the health status of the server.
func (s *server) IsAlive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.alive
}

// ServerPool holds the list of available backend servers and the current index for round-robin.
type ServerPool struct {
	servers []*server
	current uint64
}

// AddServer adds a new backend server to the pool.
func (s *ServerPool) AddServer(serverURL string) error {
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return err
	}
	
	proxy := httputil.NewSingleHostReverseProxy(parsedURL)
	proxy.Transport = &http.Transport{
		DisableKeepAlives: true,
	}

	// Initialize server as alive by default
	s.servers = append(s.servers, &server{
		URL:   parsedURL,
		Proxy: proxy,
		alive: true,
	})
	return nil
}

// GetNextPeer selects the next available server using a Round Robin algorithm.
func (s *ServerPool) GetNextPeer() *server {
	// Atomically increment the counter to get the next index in a round-robin sequence.
	nextIndex := atomic.AddUint64(&s.current, uint64(1)) - 1

	// Start searching from our round-robin index to find an alive server.
	for i := 0; i < len(s.servers); i++ {
		idx := (nextIndex + uint64(i)) % uint64(len(s.servers))
		if s.servers[idx].IsAlive() {
			return s.servers[idx]
		}
	}

	// Return nil if no servers are alive.
	return nil
}

// HealthCheck periodically checks the health of all backend servers.
func (s *ServerPool) HealthCheck() {
	for _, srv := range s.servers {
		currentServer := srv
		go func() {
			for {
				healthURL := currentServer.URL.String() + "/health"
				// Use a client with a timeout to prevent hanging.
				client := http.Client{Timeout: 5 * time.Second}
				resp, err := client.Get(healthURL)

				if err != nil || resp.StatusCode != http.StatusOK {
					// Only log if the status changes to down.
					if currentServer.IsAlive() {
						log.Printf("Backend server is down: %s", currentServer.URL)
					}
					currentServer.SetAlive(false)
				} else {
					// Only log if the status changes to up.
					if !currentServer.IsAlive() {
						log.Printf("Backend server is back up: %s", currentServer.URL)
					}
					currentServer.SetAlive(true)
				}
				if resp != nil {
					resp.Body.Close()
				}
				time.Sleep(10 * time.Second)
			}
		}()
	}
}

// loadBalancerHandler is the main handler for the load balancer.
func loadBalancerHandler(pool *ServerPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		peer := pool.GetNextPeer()
		if peer == nil {
			http.Error(w, "Service not available", http.StatusServiceUnavailable)
			return
		}

		log.Printf("Forwarding request to backend: %s", peer.URL)
		peer.Proxy.ServeHTTP(w, r)
	}
}

func main() {
	port := ":8080"
	pool := &ServerPool{}

	if err := pool.AddServer("http://localhost:9001"); err != nil {
		log.Fatalf("Failed to add backend server: %v", err)
	}
	if err := pool.AddServer("http://localhost:9002"); err != nil {
		log.Fatalf("Failed to add backend server: %v", err)
	}

	log.Printf("Configured backend servers: %s, %s", pool.servers[0].URL, pool.servers[1].URL)

	go pool.HealthCheck()

	http.HandleFunc("/", loadBalancerHandler(pool))

	log.Printf("Load balancer started, listening on http://localhost%s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start load balancer: %v", err)
	}
}
