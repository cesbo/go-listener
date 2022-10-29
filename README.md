# go-listener

Collection of listeners for TCP server

- Connlimit listener - limits concurrent connections
- TLS listener - updates certificate and key on the fly

## Quick Start

```go
ln, _ := net.Listen("tcp", "127.0.0.1:0")
ln = listener.NewConnlimitListener(ln, 1000)
ln = listener.NewTlsListener(
    ln,
    "/etc/dehydrated/certs/example.com/fullchain.pem",
    "/etc/dehydrated/certs/example.com/privkey.pem",
)
_ = s.Serve(ln)
```
