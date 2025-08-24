#!/bin/bash

# A script to compile and run all backend services concurrently.

echo "--- Forcefully cleaning up any old running services... ---"
# Use pkill with the -9 (SIGKILL) signal to be more forceful.
# The `|| true` prevents the script from exiting if no processes are found.
pkill -9 -f ./cache-server || true
pkill -9 -f ./rate-limiter || true
pkill -9 -f ./app-server || true
pkill -9 -f ./load-balancer || true
sleep 1 # Give a moment for processes to terminate.
echo "--- Cleanup complete. ---"
echo ""

echo "--- Compiling all services... ---"
go build -o cache-server cache-service.go
go build -o rate-limiter rate-limiter-service.go
go build -o app-server app-server.go
go build -o load-balancer load-balancer.go
echo "--- Compilation complete. ---"
echo ""

# Function to clean up background processes on exit
cleanup() {
    echo ""
    echo "--- Shutting down all services... ---"
    # Kill all processes that were started by this script
    kill $(jobs -p)
    echo "--- All services stopped. ---"
}

# Trap the EXIT signal to run the cleanup function when the script is terminated
trap cleanup EXIT

echo "--- Starting all services in the background... ---"

# Start the backend services
echo "Starting Cache Server..."
./cache-server &

echo "Starting Rate Limiter..."
./rate-limiter &

# Start two instances of the application server on different ports
echo "Starting App Server on port 9001..."
./app-server -port 9001 &

echo "Starting App Server on port 9002..."
./app-server -port 9002 &

# Start the main load balancer
echo "Starting Load Balancer..."
./load-balancer &

echo ""
echo "--- All services are running! ---"
echo "Load Balancer is accessible at: http://localhost:8080"
echo "Press Ctrl+C to stop all services."
echo ""

# Wait indefinitely until the script is manually stopped (e.g., with Ctrl+C)
wait