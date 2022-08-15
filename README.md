# Overview

Kooola的Channeld服务器，现阶段仅作为聊天服务器使用

# 1. 网络需求

* ## Host
	+ 外网：外网访问时通过`EXTERNAL-IP`进行访问
	+ 集群内：内网则通过Service Name或`CLUSTER-IP`来访问，推荐使用Service Name，如`kooola-channeld-chat`

* ## 端口
	+ 客户端用端口：**12108/TCP**
	+ 服务器用端口：**11288/TCP**（服务器用端口在测试环境中对外网暴露，在生产环境中对外网隐藏）
	

# 2. 开发

## main.go
用于初始化 Channeld & Master Serve，并且包含Master serve代码

## Channeld
用于消息的广播、转发和合并

以下为基于Channeld的功能性拓展
* KOOOLA_GET_USERCONNECTION：server用 Kooola AccessToken 来或取对应UE客户端的 connId
* KOOOLA_PRIVATE_CHAT：使用消息转发来实现用户私聊

## Master Server
在启动 Channeld 后会通过 client.go 作为 server 连接到 Channeld

Master server 会拥有 Global channel 并订阅 Channeld 的 auth 事件，当其他 server 连接 Channeld 并 auth 成功后，会为其订阅 Global channel

## auth.go
提供 client 登录校验、记录 Kooola 用户 Kooola AccessToken 对应 ConnId 的映射功能。

client 尝试连接时需要提供 pit(kooola Username) 和 lt(kooola AccessToken)，通过访问 APIServer 来验证是否为合法 client，若合法则将 AccessToken 和 connId 关系记录下，之后 UE Server 可以通过 AccessToken 获取到此客户端的connId
# 3. 配置Channeld地址

* 在UEServer的k8s.yaml中添加环境变量：`CHANNELD_ADDR=<KooolaChanneldHost>:<ServerPort>`
	+ `<KooolaChanneldHost>`为`kooola-channeld-chat[-server]`，**生产环境需要带`-server`**
	+ `<ServerPort>`默认为11288
For example:
```
kind: Deployment
metadata:
  ...
spec:
  ...
  template:
    ...
    spec:
      containers:
      - name: my-app
        env:
        - name: CHANNELD_ADDR
          value: kooola-channeld-chat-server:11288
...
```

*  UEClient的DefaultGame.ini文件中添加配置项
	* `<KooolaChanneldIP>`为`service/kooola-channeld-chat`分配的`EXTERNAL-IP`
	* `<ClientPort>`默认为12108
```
[/Script/KooolaMusic.ChanneldGameInstanceSubsystem]
ChanneldAddrConfiguration=<KooolaChanneldIP>:<ClientPort>
```



# 4. 部署

## 1. 构建docker镜像

执行 `./cmd/BuildDockerImage.bat` 构建docker镜像到本地

若想测试构建的docker，执行 `docker run --rm -d --name kooola-channeld-chat -p 11288:11288/tcp -p 12108:12108/tcp kooola/kooola-channeld-chat`

## 2. 上传docker镜像

执行 `./cmd/UploadDockerImage.bat [VERSION]` 上传docker镜像到到阿里云镜像仓库，默认版本号为latest

## 3. 部署到阿里云K8s

部署时需要做出生产环境和测试环境的区分，主要体现在创建Service时使用的配置不同

* 修改**k8s/<testing|production>/channeld-deployment.yaml**中的`spec.template.spec.containers.image`的版本号为[上传时](#2-上传docker镜像)设置的版本号
* 执行 `kubectl apply -f ./k8s/<testing|production>` 使阿里云k8s从镜像仓库拉取镜像，并创建Deployment和Service

## 4. 更新
1. [构建docker镜像](#1-构建docker镜像)
2. 设置版本号并[上传docker镜像](#2-上传docker镜像)
3. 修改**k8s/<testing|production>/channeld-deployment.yaml**中的`spec.template.spec.containers.image`的版本号为[上传时](#2-上传docker镜像)设置的版本号
4. 执行 `kubectl <apply|replace --force> -f ./k8s/<testing|production>/channeld-deployment.yaml` 更新K8s的Deployment。若上传镜像版本号没有改变则使用 `replace --force` 正常情况下使用 `apply`

# 5. 运维

* K8s中所有的内容`kubectl get all -A`

* 只查询channeld相关`kubectl get all -A --field-selector metadata.namespace=channeld`

* 打印服务的日志`kubectl logs service/kooola-channeld-chat -n channeld --tail 30 -f` 

* ssh`kubectl exec -it service/kooola-channeld-chat -n channeld -- bash` 
