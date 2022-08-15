@echo off

set TAG=%1
if "%TAG%"=="" set TAG=latest

docker logout

docker login kooola-registry.ap-southeast-1.cr.aliyuncs.com --username=realydao@gmail.com --password=Metaw0rkin9#TheHouse

docker tag kooola/kooola-channeld-chat kooola-registry.ap-southeast-1.cr.aliyuncs.com/kooola/channeld-chat:%TAG%

docker push kooola-registry.ap-southeast-1.cr.aliyuncs.com/kooola/channeld-chat:%TAG%