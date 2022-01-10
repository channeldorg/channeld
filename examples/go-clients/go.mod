module channeld.clewcat.com/examples/client

go 1.16

require (
	channeld.clewcat.com/channeld v0.0.0-00010101000000-000000000000
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gorilla/websocket v1.4.2
	google.golang.org/protobuf v1.27.1
)

replace channeld.clewcat.com/channeld => ../../../channeld
