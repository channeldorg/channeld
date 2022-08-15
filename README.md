# Overview

Kooola的Channeld服务器，现阶段仅作为聊天服务器使用

# 1. 端口需求
* 客户端用端口：**12108/TCP**
* 服务器用端口：**11288/TCP**

# 2. 开发

程序由主程、Master Server和Channeld组成
* 主程提供客户端登录校验、记录Kooola用户体系对应ConnId的映射等
* Master Server负责管理Global频道

# 3. 配置
* k8s 环境变量：`CHANNELD_ADDR=<KooolaChanneldHost>:<ServerPort>`
	+ `<KooolaChanneldHost>`为`kooola-channeld-chat[-server]`，**生产环境需要带`-server`**
	+ `<ServerPort>`默认为11288

* Kooola启动器command line参数：`-channeldAddr=<KooolaChanneldIP>:<ClientPort>`
	+ `<KooolaChanneldIP>`为`service/kooola-channeld-chat`分配的`EXTERNAL-IP`
	+ `<ClientPort>`默认为12108

# 4. 部署

## 1. 构建docker镜像

执行 `./cmd/DockerBuildAndRun.bat` 构建docker镜像到本地

若想测试构建的docker，执行 `docker run --rm -d --name kooola-channeld-chat -p 11288:11288/tcp -p 12108:12108/tcp kooola/kooola-channeld-chat`

## 2. 上传docker镜像

执行 `./cmd/UploadImage.bat <VERSION>` 上传docker镜像到到阿里云镜像仓库，默认版本号为latest

## 3. 部署到阿里云K8s

部署时需要做出生产环境和测试环境的区分，主要体现在创建Service时使用的配置不同

* ### 创建Deployment
	+ 修改**channeld-deployment.yaml**中的`spec.template.spec.containers.image`的版本号为[上传时](#2-上传docker镜像)设置的版本号
	+ 执行 `kubectl apply -f ./k8s/channeld-deployment.yaml` 使阿里云k8s从镜像仓库拉取镜像，并创建Deployment

* ### 创建Service
	+ **测试环境：** Service会把client和server端口都暴露到外网
`kubectl apply -f ./k8s/channeld-service-dev.yaml`

	+ **生产环境：** Service只会暴露client端口到外网
`kubectl apply -f ./k8s/channeld-service-prod.yaml`

## 4. 更新
* [构建docker镜像](#1-构建docker镜像)
* 设置版本号并[上传docker镜像](#2-上传docker镜像)
* 修改channeld-deployment.yaml中的spec.template.spec.containers.image的版本号为上传时设置的版本号
* 执行 `kubectl replace --force -f ./k8s/channeld-deployment.yaml` ，替换K8s的Deployment

# 5. 运维

* K8s中所有的内容`kubectl get all -A`

* 只查询channeld相关`kubectl get all -A --field-selector metadata.namespace=channeld`

* 打印服务的日志`kubectl logs service/kooola-channeld-chat -n channeld --tail 30 -f` 

* ssh`kubectl exec -it service/kooola-channeld-chat -n channeld -- bash` 
