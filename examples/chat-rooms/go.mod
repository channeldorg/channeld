module github.com/channeldorg/channeld/examples/chat-rooms

go 1.25

require (
	github.com/channeldorg/channeld v0.0.0
	github.com/prometheus/client_golang v1.11.1
	github.com/stretchr/testify v1.8.1
	google.golang.org/protobuf v1.28.1
)

replace github.com/channeldorg/channeld => ../..
