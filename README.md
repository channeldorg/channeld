# Overview
channeld is an efficient gateway service designed for massive online systems with high fidelity interactions, such as multiplayer games and 3D virtual events. It can be seamlessly integrated with the mainstream game engines (e.g. Unity, Unreal Engine) to develop dedicated servers, together to distributionally simulate the parallel game sessions or seamless open worlds.

channeld是为大型多人交互系统（如MMO，虚拟演唱会）设计的高性能网关服务，可无缝接入UE、Unity等引擎进行专用服务器的开发，用于分布式地模拟多房间及无缝大世界等应用。

<img alt="architecture" height="360" src="doc/architecture.png"/>

See the concepts in the [design doc](doc/design.md).

## Applications 应用场景
There are three major types of application benifit from channeld's architecture design:

channeld的架构设计可应用于以下三种场景：
### 1.Relay Servers 中继服务器

<img height="300" src="doc/relay.png"/>

channeld can be used as the relay server to forward/broadcast messages between game clients.

channeld可用于中继服务器，用于转发/广播游戏客户端之间的消息。

### 2.Dedicated Server Gateway 专用服务器网关

<img height="360" src="doc/dedicated.png"/>

channeld can be used as the gateway server to route messages to different dedicated servers.

channeld可用于专用服务器网关，用于将消息路由到不同的专用服务器。

### 3.Seamless Distributed Server Gateway 无缝分布式服务器网关

<img height="400" src="doc/seamless.png"/>

The ultimate purpose of channeld is to enable distributed composition of dedicated servers, together to form a seamless large virtual world.

channeld的最终目标是实现专用服务器的分布式组合，从而形成无缝的大型虚拟世界。

## Key features:
* Protobuf-based binary protocol over TCP, KCP or WebSocket
* FSM-based message filtering
* Fanout-based data pub/sub of any type defined with Protobuf
* Interest management based on channel and data pub/sub
* Integration with the mainstream game engines ([Unity](https://github.com/channeldorg/channeld-unity-mirror), [Unreal Engine](https://github.com/channeldorg/channeld-ue-plugin))
* [WIP] Backend servers load-balancing with auto-scaling

关键特性：
* 基于Protobuf的二进制协议，支持TCP、KCP、WebSocket
* 基于有限状态机的消息过滤
* 基于扇出的数据发布/订阅，支持任意Protobuf定义的数据类型
* 基于频道和数据发布/订阅的兴趣管理
* 接入主流游戏引擎（[Unity](https://github.com/channeldorg/channeld-unity-mirror), [Unreal Engine](https://github.com/channeldorg/channeld-ue-plugin))

## Performance 性能
channeld is aimed to support 10K connections and 100K mps(messages per second) on a single node (uplink + downlink), and 10M+ mps in a distributed system.

channeld的目标是在单个节点（上行+下行）上支持10K连接和100K mps（每秒消息数），以及在分布式系统中支持10M+ mps。

## Roadmap 路线图
There is a [dedicated roadmap documentation](doc/roadmap.md).

Keep in mind that the requirements of the real-life projects will decide the priority of the development.

路线图详见[这里](doc/roadmap.md)。

请注意，实际项目的需求将决定开发的优先级。
# Getting Started
## 1. Clone the source code
## 2. Docker
The fastest way to run the server is with [Docker Compose](https://docs.docker.com/compose/).

There's a [docker-compose file](docker-compose.yml) set up for running the chat rooms demo. Navigate to the root of the repo and run the command:

`docker-compose up chat`

## 3. The chat rooms demo
After starting the server, browse to http://localhost:8080.

Use the input box at the bottom to send messages, to the GLOBAL channel by default. The input box can also be used to send commands, which are started with '/'. The supported commands are:

* `list [typeFilter] [metadataFilter1],[metadataFilter2],...` // the result format is `Channel(<ChannelTypeName> <ChannelId>)`
* `create <channelType> <metadata>` // the channelType is an integer. See the ChannelType enum value defined in [the proto](pkg/channeldpb/channeld.proto) file
* `remove <channelId>` // only the channel creator has the permission to remove the channel.
* `sub <channelId>` // subscribe to the channel
* `unsub <channelId>` // unsubscribe from the channel
* `switch <channelId>` // switch the active channel. Only the active channel displays the new chat messages.

## 4. The Unity tank demo
Follow these steps if the docker image has not been built for the tanks service yet:
1. Check out the [unity-mirror-channeld](https://github.com/channeldorg/channeld-unity-mirror) repo
2. Create the Unity project following the [instruction](https://github.com/channeldorg/channeld-unity-mirror#how-to-run-the-tank-demo)
3. Either build the Linux player from Unity Editor (Build -> Linux Server), or via the command: `Unity -batchmode -nographics -projectPath <PATH_TO_YOUR_UNITY_PROJECT> -executeMethod BuildScript.BuildLinuxServer -logFile build.log -quit`. The path to the Unity Editor needs to added to the PATH environment argument in order to run the command.
4. Build the docker image: `docker build -t channeld/tanks .`

Running the Unity tanks demo with Docker is similar to running the chat rooms demo. Navigate to the root of the repo and run the command:

`docker-compose up tanks`

Then you can the play the game in Unity Editor. See the [full instruction here](https://github.com/channeldorg/channeld-unity-mirror#how-to-run-the-tank-demo).