module channeld.clewcat.com/channeld/examples/sim-clients

go 1.16

require (
	channeld.clewcat.com/channeld v0.0.0-00010101000000-000000000000
	channeld.clewcat.com/channeld/examples/chat-rooms v0.0.0
	channeld.clewcat.com/channeld/examples/unity-mirror-tanks v0.0.0
	github.com/prometheus/client_golang v1.11.1
	google.golang.org/protobuf v1.27.1
)

replace channeld.clewcat.com/channeld => ../..

replace channeld.clewcat.com/channeld/examples/chat-rooms => ../../examples/chat-rooms

replace channeld.clewcat.com/channeld/examples/unity-mirror-tanks => ../../examples/unity-mirror-tanks
