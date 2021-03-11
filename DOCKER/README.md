## DOCKER
在Docker中运行
```
docker pull golang:1.14
docker pull alpine:latest
docker build -t chainmaker:v0.5.0 -f ./DOCKER/Dockerfile .
docker run -it chainmaker:v0.5.0
```
其中，`0.5.0`是对应版本号，可以修改。
