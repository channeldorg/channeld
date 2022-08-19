# Core features
- [x] Channel pub/sub
- [x] Data update and fan-out
- [x] FSM-based message filtering
- [x] Message broadcasting
- [x] Authentication
- [ ] Channel ACL
- [ ] DDoS Protection
- [ ] Health check
- [ ] Front-end load-balancing
- [x] Spatial-based pub/sub
- [ ] Spatial-based load-balancing

# Modules
- [x] Stub(RPC) support
- [x] WebSocket support
- [x] KCP support
- [x] [Snappy](https://github.com/golang/snappy) compression
- [ ] [Markov-chain](https://en.wikipedia.org/wiki/Markov_chain) compression
- [ ] Encryption
- [ ] Replay
- [x] Prometheus integration

# Optimizations
- [x] Read/write the packet using Protobuf
- ~~[ ] Use [gogoprotobuf](https://github.com/gogo/protobuf) for faster marshalling/unmarshalling~~
- [x] Enable custom merge of channel data messages

# Tests
- [ ] Unit tests
- [ ] Smoke tests
- [ ] Scale tests

# SDKs
- [ ] Javascript SDK
- [x] Unity C# SDK
- [ ] Unreal C++ SDK

# Tools
- [x] Simulated client (Go)
- [ ] Web inspector

# Example projects
- [x] Web chat rooms
    - [x] Implement the Javascript client library
    - [x] Implement the commands
    - [ ] Scale test with 10K connections
    - [ ] Complete the UI
- [ ] Unity tank game
    - [x] Implement the C# client library
    - [ ] Mirror Integration
        - [x] Transport
        - [x] SyncVar and NetworkTransform
        - [x] Observers and Interest Management
        - [ ] SyncVar and RPC code generation
    - [x] Multi-server support
- [ ] Unreal seamless world travelling
    - [x] Implement the C++ client library
    - [x] Blueprint support
    - [x] Integrate with Unreal's networking stack
    - [ ] Integrate with Unreal's Replication system
    - [ ] Replication codegen
    - [x] Multi-server support
    - [ ] KCP support
    - [ ] Data traffic compression
- [ ] Dynamic region load-balancing
