#!/bin/bash

# A script to automatically test all features of the distributed backend system.

# Define some colors for nice output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}--- Starting Automated System Test ---${NC}"
echo "Watch the server logs in your other terminal to see the system in action."
echo ""
sleep 2

# --- Test 1: Load Balancing and Caching ---
echo -e "${YELLOW}--- Test 1: Verifying Load Balancing (Round Robin) & Caching ---${NC}"

echo "Sending Request 1 for user 101... (Expect Cache MISS on first server)"
curl -v http://localhost:8080/users/101
echo ""
sleep 2

echo "Sending Request 2 for user 101... (Expect Cache MISS on second server)"
curl -v http://localhost:8080/users/101
echo ""
sleep 2

echo "Sending Request 3 for user 101... (Expect Cache HIT on first server - should be faster!)"
curl -v http://localhost:8080/users/101
echo ""
echo -e "${GREEN}--> Checkpoint: Did the requests alternate between servers in the logs? Did the third request show a HIT?${NC}"
sleep 4

# --- Test 2: Health Checks and Fault Tolerance ---
echo -e "${YELLOW}--- Test 2: Verifying Health Checks & Fault Tolerance ---${NC}"

# Find the PID of the server running on port 9001
PID_TO_KILL=$(pgrep -f "./app-server -port 9001")

if [ -z "$PID_TO_KILL" ]; then
    echo "Could not find PID for server on port 9001. Skipping fault tolerance test."
else
    echo "Found app-server on port 9001 with PID: $PID_TO_KILL"
    echo "Killing server on port 9001 to simulate a failure..."
    kill $PID_TO_KILL
    sleep 2
    
    echo "Waiting 12 seconds for the load balancer's health check to detect the failure..."
    echo "(Watch the server logs for a 'Backend server is down' message)"
    sleep 12

    echo "Sending new requests. All traffic should now be routed to the healthy server on port 9002."
    curl -v http://localhost:8080/users/202
    echo ""
    sleep 1
    curl -v http://localhost:8080/users/203
    echo ""

    echo -e "${GREEN}--> Checkpoint: Did the logs show server 9001 went down? Are all new requests going to 9002?${NC}"
fi
sleep 4

# --- Test 3: Rate Limiting ---
echo -e "${YELLOW}--- Test 3: Verifying the Rate Limiter ---${NC}"
echo "Sending a burst of 50 requests to user 999..."
echo "The first ~20 should succeed, then we should see 'Too Many Requests'."
echo ""

for i in {1..50}; do
    # Use -s for silent mode to keep the output clean
    curl -s -o /dev/null -w "Request $i: %{http_code}\n" http://localhost:8080/users/999
    # sleep 0.05 # Sleep for a very short time between requests
done

echo ""
echo -e "${GREEN}--> Checkpoint: Did the requests start failing with a 429 status code after the initial burst?${NC}"
sleep 2

echo ""
echo -e "${BLUE}--- Automated System Test Complete ---${NC}"
echo "You can now press Ctrl+C in the other terminal to stop all services."

