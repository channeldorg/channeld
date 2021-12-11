~#!/bin/bash

# also could use this, hasn't verified
# minikube start --image-mirror-country='cn'

minikube start --image-mirror-country='cn' --image-repository='registry.cn-hangzhou.aliyuncs.com/google_containers' --profile agones
minikube start \
	--registry-mirror=https://registry.docker-cn.com \
	--image-repository=registry.cn-hangzhou.aliyuncs.com/google_containers \
	--vm-driver=docker \
	--alsologtostderr -v=8 \
	--kubernetes-version v1.21.5 \
	--profile agones \
	--base-image registry.cn-hangzhou.aliyuncs.com/google_containers/kicbase:v0.0.28

