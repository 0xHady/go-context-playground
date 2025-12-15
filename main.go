package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/0xHady/go-context-playground/contextdemo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

// ServerB implements the protobuf-defined service and honors context cancellation.
type ServerB struct {
	contextdemo.UnimplementedServerBServer
}

func doDummyHttpCall(ctx context.Context) (*http.Response, error) {
	client := &http.Client{
		Timeout: 10 * time.Second, // safety net (still respects ctx)
	}

	body := `{
	"apiKey": "dxxxx=="
}`

	reqHTTP, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"sdfaodododododododin",
		strings.NewReader(body),
	)
	if err != nil {
		log.Printf("Server B: failed to create HTTP request: %v", err)
		return nil, err
	}

	reqHTTP.Header.Set("accept", "text/plain")
	reqHTTP.Header.Set("Content-Type", "application/json")
	reqHTTP.Header.Set(
		"d",
		"df.d.d",
	)

	resp, err := client.Do(reqHTTP)
	if err != nil {
		log.Printf("Server B: HTTP call failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("Server B: HTTP call completed with status %s", resp.Status)

	return resp, nil
}

func (s *ServerB) Process(ctx context.Context, req *contextdemo.ContextRequest) (*contextdemo.ContextResponse, error) {
	log.Printf("Server B: Received request: %s", req.Payload)
	log.Println("Server B: Starting 5-second sleep...")

	select {
	case <-time.After(5 * time.Second):
		log.Println("Server B: Finished processing after 5 seconds")
		filename := "server_b_completed.txt"
		if err := os.WriteFile(filename, []byte("Server B has completed processing."), 0644); err != nil {
			log.Printf("Server B: Error writing completion file: %v", err)
			return nil, err
		}
		res, err := doDummyHttpCall(context.Background())
		if err != nil {
			log.Printf("Server B: Error during dummy HTTP call: %v", err)
			return nil, err
		}
		log.Printf("Server B: Dummy HTTP call response status: %s", res.Status)

		return &contextdemo.ContextResponse{Message: "Server B processed: " + req.Payload}, nil
		// case <-ctx.Done():
		// 	log.Printf("Server B: Context cancelled before completion: %v", ctx.Err())
		// 	return nil, status.FromContextError(ctx.Err()).Err()
	}
}

// ServerA forwards the call to Server B using the same context.
type ServerA struct {
	contextdemo.UnimplementedServerAServer
	serverBAddress string
}

func (s *ServerA) Handle(ctx context.Context, req *contextdemo.ContextRequest) (*contextdemo.ContextResponse, error) {
	log.Printf("Server A: Received request: %s", req.Payload)
	log.Println("Server A: Calling Server B with propagated context...")

	conn, err := grpc.DialContext(ctx, s.serverBAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Server A: Error connecting to Server B: %v", err)
		return nil, err
	}
	defer conn.Close()

	// operation 1
	twoSecCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	cancel()

	client := contextdemo.NewServerBClient(conn)
	resp, err := client.Process(twoSecCtx, req)
	if err != nil {
		log.Printf("Server A: Error calling Server B: %v", err)
		return nil, err
	}

	select {
	case <-twoSecCtx.Done():
		log.Printf("CCCC: Server A: Context cancelled before receiving reply from Server B: %v", ctx.Err())
	default:
		// log.Println("Server A: Successfully received reply from Server B")
	}

	log.Printf("Server A: Received reply from Server B: %s", resp.Message)
	return &contextdemo.ContextResponse{Message: "Server A forwarded to Server B: " + resp.Message}, nil
}

func startServerB(port string) {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Server B: Failed to listen on %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	contextdemo.RegisterServerBServer(grpcServer, &ServerB{})
	reflection.Register(grpcServer)

	log.Printf("Server B: Listening on %s", port)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Server B: Serve error: %v", err)
		}
	}()
}

func startServerA(port string, serverBAddress string) {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Server A: Failed to listen on %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	contextdemo.RegisterServerAServer(grpcServer, &ServerA{serverBAddress: serverBAddress})
	reflection.Register(grpcServer)

	log.Printf("Server A: Listening on %s", port)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Server A: Serve error: %v", err)
	}
	// func() {
	//
	// }()
}

func main() {
	serverBPort := ":50056"
	serverAPort := ":50055"
	serverBAddress := "0.0.0.0:50056"

	// Clean up old completion file from previous runs.
	_ = os.Remove("server_b_completed.txt")

	// Start Server B
	go startServerB(serverBPort)

	// Give Server B time to start
	time.Sleep(500 * time.Millisecond)

	// Start Server A
	startServerA(serverAPort, serverBAddress)

	// Give Server A time to start
	time.Sleep(500 * time.Millisecond)

	// Now make a client request to Server A with a 2-second timeout
	log.Println("\n=== Client: Making request to Server A with 2-second timeout ===")
	//
	// conn, err := grpc.Dial("localhost"+serverAPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	// if err != nil {
	// 	log.Fatalf("Client: Failed to connect to Server A: %v", err)
	// }
	// defer conn.Close()
	// client := contextdemo.NewServerAClient(conn)
	//
	// // Create a context with 2-second timeout
	// ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	// defer cancel()
	//
	// var reply string
	// callDone := make(chan error, 1)
	//
	// // Make the call in a goroutine so we can respect the context timeout
	// go func() {
	// 	resp, err := client.Handle(ctx, &contextdemo.ContextRequest{Payload: "test-request"})
	// 	if err == nil && resp != nil {
	// 		reply = resp.Message
	// 	}
	// 	callDone <- err
	// }()
	//
	// // Wait for either the call to complete or the context to timeout
	// select {
	// case err := <-callDone:
	// 	if err != nil {
	// 		log.Printf("Client: RPC call failed: %v", err)
	// 	} else {
	// 		log.Printf("Client: Received reply: %s", reply)
	// 	}
	// 	// return
	// case <-ctx.Done():
	// 	log.Printf("Client: Context cancelled (timeout after 2 seconds): %v", ctx.Err())
	// 	// log.Println("Client: Note - With propagated context, Server B should cancel too")
	// 	// cancel()
	// 	// return
	// }
	//
	// // Wait additional time to see if Server B completes
	// log.Println("\nClient: Waiting 4 more seconds to see if Server B completes...")
	// time.Sleep(4 * time.Second)
	//
	// if _, err := os.Stat("server_b_completed.txt"); err == nil {
	// 	log.Println("Client: Server B completion file FOUND (unexpected when context is propagated)")
	// } else if errors.Is(err, os.ErrNotExist) {
	// 	log.Println("Client: Server B completion file NOT found (expected with propagated context)")
	// } else {
	// 	log.Printf("Client: Error checking completion file: %v", err)
	// }
	//
	// log.Println("\n=== Demonstration Complete ===")
	// log.Println("Expected behavior:")
	// log.Println("1. Client request times out after 2 seconds")
	// log.Println("2. Propagated context cancels Server B before it completes the 5-second work")
	// log.Println("3. No completion file should be written by Server B when cancellation propagates")
}
