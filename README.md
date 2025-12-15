# go-context-playground

A demonstration of Go context propagation behavior with RPC servers.

## Scenario

This project demonstrates what happens when context is not properly propagated between RPC servers:

- **Server A**: Receives a client request with a 2-second timeout
- **Server B**: Performs work that takes 5 seconds
- **Key Issue**: Server A calls Server B using `context.Background()` instead of propagating the original context

## Expected Behavior

1. The client request times out after 2 seconds
2. Server B continues processing and completes after 5 seconds (despite the client timeout)
3. This demonstrates that using `context.Background()` breaks context propagation, allowing Server B to continue processing even though the original request context was cancelled

## Running the Demo

```bash
go run main.go
```

## Output

The program will show:
- Server B starting its 5-second processing
- Client timeout after 2 seconds
- Server B completing its work after 5 seconds total

This illustrates the importance of properly propagating context in distributed systems to respect cancellation signals and timeouts.