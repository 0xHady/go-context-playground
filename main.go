package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"time"
)

// ServerB represents the RPC server that sleeps for 5 seconds
type ServerB struct{}

// ProcessRequest is the RPC method that sleeps for 5 seconds
func (s *ServerB) ProcessRequest(req string, reply *string) error {
	log.Printf("Server B: Received request: %s", req)
	log.Println("Server B: Starting 5-second sleep...")
	
	// Sleep for 5 seconds
	time.Sleep(5 * time.Second)
	
	log.Println("Server B: Finished processing after 5 seconds")
	*reply = fmt.Sprintf("Server B processed: %s", req)
	return nil
}

// ServerA represents the RPC server that calls Server B
type ServerA struct {
	serverBAddress string
}

// HandleRequest is the RPC method that calls Server B with context.Background()
func (s *ServerA) HandleRequest(req string, reply *string) error {
	log.Printf("Server A: Received request: %s", req)
	
	// Call Server B with a Background context (not propagating the original context)
	log.Println("Server A: Calling Server B with context.Background()...")
	
	client, err := rpc.Dial("tcp", s.serverBAddress)
	if err != nil {
		log.Printf("Server A: Error connecting to Server B: %v", err)
		return err
	}
	defer client.Close()
	
	var serverBReply string
	// Using context.Background() here - not propagating the original context
	err = client.Call("ServerB.ProcessRequest", req, &serverBReply)
	if err != nil {
		log.Printf("Server A: Error calling Server B: %v", err)
		return err
	}
	
	log.Printf("Server A: Received reply from Server B: %s", serverBReply)
	*reply = fmt.Sprintf("Server A forwarded to Server B: %s", serverBReply)
	return nil
}

func startServerB(port string) {
	serverB := new(ServerB)
	rpc.Register(serverB)
	
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Server B: Failed to listen on %s: %v", port, err)
	}
	
	log.Printf("Server B: Listening on %s", port)
	
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Server B: Accept error: %v", err)
				continue
			}
			go rpc.ServeConn(conn)
		}
	}()
}

func startServerA(port string, serverBAddress string) {
	serverA := &ServerA{serverBAddress: serverBAddress}
	rpc.Register(serverA)
	
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Server A: Failed to listen on %s: %v", port, err)
	}
	
	log.Printf("Server A: Listening on %s", port)
	
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Server A: Accept error: %v", err)
				continue
			}
			go rpc.ServeConn(conn)
		}
	}()
}

func main() {
	serverBPort := ":50056"
	serverAPort := ":50055"
	serverBAddress := "localhost:50056"
	
	// Start Server B
	startServerB(serverBPort)
	
	// Give Server B time to start
	time.Sleep(500 * time.Millisecond)
	
	// Start Server A
	startServerA(serverAPort, serverBAddress)
	
	// Give Server A time to start
	time.Sleep(500 * time.Millisecond)
	
	// Now make a client request to Server A with a 2-second timeout
	log.Println("\n=== Client: Making request to Server A with 2-second timeout ===")
	
	client, err := rpc.Dial("tcp", "localhost"+serverAPort)
	if err != nil {
		log.Fatalf("Client: Failed to connect to Server A: %v", err)
	}
	defer client.Close()
	
	// Create a context with 2-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	var reply string
	callDone := make(chan error, 1)
	
	// Make the call in a goroutine so we can respect the context timeout
	go func() {
		callDone <- client.Call("ServerA.HandleRequest", "test-request", &reply)
	}()
	
	// Wait for either the call to complete or the context to timeout
	select {
	case err := <-callDone:
		if err != nil {
			log.Printf("Client: RPC call failed: %v", err)
		} else {
			log.Printf("Client: Received reply: %s", reply)
		}
	case <-ctx.Done():
		log.Printf("Client: Context cancelled (timeout after 2 seconds): %v", ctx.Err())
		log.Println("Client: Note - Server B should still continue processing despite client timeout")
	}
	
	// Wait additional time to see if Server B completes
	log.Println("\nClient: Waiting 4 more seconds to see if Server B completes...")
	time.Sleep(4 * time.Second)
	
	log.Println("\n=== Demonstration Complete ===")
	log.Println("Expected behavior:")
	log.Println("1. Client request times out after 2 seconds")
	log.Println("2. Server B continues processing and completes after 5 seconds total")
	log.Println("3. This proves that using context.Background() in Server A breaks context propagation")
}
