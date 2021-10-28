# Core features
- [x] Channel sub/unsub
- [x] Data update and fan-out
- [x] FSM-based message filtering
- [x] Message broadcasting
- [ ] Authentication
- [ ] Spatial-based sub/unsub
- [ ] Spatial-basd load-balancing

# Modules
- [x] Stub(RPC) support
- [x] WebSocket support
- [ ] KCP support
- [ ] [Snappy](https://github.com/golang/snappy) compression support

# Optimizations
- [x] Read/write the packet using Protobuf
- [ ] Use [gogoprotobuf](https://github.com/gogo/protobuf) for faster marshalling/unmarshalling

# Tests
- [.] Unit tests
- [ ] Smoke tests
- [ ] Scale tests

# SDKs
- [ ] Javascript SDK
- [ ] Unity SDK
- [ ] Unreal SDK

# Example projects
- [.] Chat rooms
    - [x] Implement the Javascript framework
    - [.] Implement the commands
    - [ ] Scale test with 10K connections
    - [ ] Implement the UI
- [ ] Seamless world travelling
- [ ] Dynamic region load-balancing
- [ ] Relay server