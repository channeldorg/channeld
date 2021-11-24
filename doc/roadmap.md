# Core features
- [x] Channel pub/sub
- [x] Data update and fan-out
- [x] FSM-based message filtering
- [x] Message broadcasting
- [ ] Authentication
- [ ] Health check
- [ ] Front-end load-balancing
- [ ] Spatial-based pub/sub
- [ ] Back-end load-balancing

# Modules
- [x] Stub(RPC) support
- [x] WebSocket support
- [ ] KCP support
- [ ] [Snappy](https://github.com/golang/snappy) compression
- [ ] [Markov-chain](https://en.wikipedia.org/wiki/Markov_chain) compression
- [ ] Encryption
- [x] Prometheus integration

# Optimizations
- [x] Read/write the packet using Protobuf
- [ ] Use [gogoprotobuf](https://github.com/gogo/protobuf) for faster marshalling/unmarshalling

# Tests
- [.] Unit tests
- [ ] Smoke tests
- [ ] Scale tests

# SDKs
- [ ] Web SDK
- [ ] Unity SDK
- [ ] Unreal SDK

# Example projects
- [.] Web chat rooms
    - [x] Implement the Javascript client library
    - [x] Implement the commands
    - [.] Scale test with 10K connections
    - [ ] Complete the UI
- [ ] Unity relay server
    - [ ] Implement the C# client library
    - [ ] Implement the reflection-based schema
    - [ ] Implement the network replication
- [ ] Unreal seamless world travelling
    - [ ] Implement the C++ client library
    - [ ] Integrate with Unreal's networking stack
    - [ ] ...
- [ ] Dynamic region load-balancing
