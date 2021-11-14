channeld is a light-weight and efficient **messaging gateway** server designed for **distributed game servers** (typically MMO) 
and other backend applications that require real-time, subscription-based user interaction with high concurrency (e.g. an instance messenger).

## Features:
* Protobuf-based binary protocol over TCP, KCP or WebSocket
* Load-balancing backend servers with auto-scaling
* FSM-based message filtering
* Channel-based data subscription and storage of any type defined with Protocol Buffer
* Area of interest management based on channel and data subscriptions

## Binary protocol:
-----------------------------------------------------------------------------------
CHNL(4B) | ChannelID(4B) | StubID(4B) | MessageType(4B) | BodySize(4B) | MessageBody
------------------------------------------------------------------------------------
1. The first 4 bytes is the indentifier of a valid channeld packet ("CHNL" short for channel).
2. The second 4 bytes is an uint32 indicates which channel the message is sent to. 0 means the message should be handled in the global channel (e.g. a CreateChannel message).
3. The third 4 bytes is an uint32 indentifier as the RPC stub. 0 means the packet is not a RPC message.
4. Followed by the 4 bytes as an uint32 indicates the [message type](proto/message_types.proto).
5. Followed by the 4 bytes as the uint32 size of the remaining message body.
6. The remaining bytes is the message body, which is the marshalled data of a protobuf message.


## Performance
channeld is aimmed to support 10Ks connections and 1Ms packets per second on a single node, and 100Ms packets in a distributed system.

## Examples:
* Chat rooms
HTML + JavaScript, WebSocket
* Relay server
Unity, TCP
* Seamless world travelling
Unreal, KCP
* Dynamic region load-balancing
