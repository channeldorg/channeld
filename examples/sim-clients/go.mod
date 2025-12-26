module github.com/channeldorg/channeld/examples/sim-clients

go 1.25

require (
	github.com/channeldorg/channeld v0.0.0
	github.com/channeldorg/channeld/examples/chat-rooms v0.0.0
	github.com/channeldorg/channeld/examples/unity-mirror-tanks v0.0.0
	google.golang.org/protobuf v1.28.1
)

replace github.com/channeldorg/channeld => ../..

replace github.com/channeldorg/channeld/examples/chat-rooms => ../../examples/chat-rooms

replace github.com/channeldorg/channeld/examples/unity-mirror-tanks => ../../examples/unity-mirror-tanks
