# TH-08: Connection Pooling

## Overview

Maintain a pool of IMAP connections for concurrent operations.

## Technical Requirements

```go
type ConnectionPool struct {
    connections chan *IMAPConnection
    maxSize     int
    factory     func() (*IMAPConnection, error)
}

func (p *ConnectionPool) Get(ctx context.Context) (*IMAPConnection, error) {
    select {
    case conn := <-p.connections:
        if conn.IsAlive() {
            return conn, nil
        }
        // Connection dead, create new
        return p.factory()
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Pool empty, create new if under limit
        return p.factory()
    }
}

func (p *ConnectionPool) Put(conn *IMAPConnection) {
    if !conn.IsAlive() {
        conn.Close()
        return
    }
    select {
    case p.connections <- conn:
    default:
        // Pool full, close connection
        conn.Close()
    }
}
```

## Acceptance Criteria

- [ ] Pool of reusable connections
- [ ] Automatic reconnection
- [ ] Concurrent operations supported
- [ ] Resource limits enforced

## Estimated Complexity

Medium
