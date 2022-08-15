@echo off

cd "%~dp0..\"

docker build -f Dockerfile -t kooola/kooola-channeld-chat .

docker stop kooola-channeld-chat

docker run --rm -d --name kooola-channeld-chat -p 11288:11288/tcp -p 12108:12108/tcp kooola/kooola-channeld-chat