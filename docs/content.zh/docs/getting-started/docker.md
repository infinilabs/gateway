---
weight: 35
title: 容器部署
asciinema: true
---

# 容器部署

极限网关支持容器方式部署。

## 安装演示

{{< asciinema key="/gateway_on_docker" autoplay="1"  start-at="49" rows="30" preload="1" >}}

## 下载镜像

极限网关的镜像发布在 Docker 的官方仓库，地址如下：

[https://hub.docker.com/r/infinilabs/gateway](https://hub.docker.com/r/infinilabs/gateway)

使用下面的命令即可获取最新的容器镜像：

```
docker pull infinilabs/gateway:{{< globaldata "gateway" "version" >}}
```

## 验证镜像

将镜像下载到本地之后，可以看到极限网关的容器镜像非常小，只有不到 25MB，所以下载的速度应该是非常快的。

```
✗ docker images |grep "gateway" |grep "{{< globaldata "gateway" "version" >}}"
REPOSITORY                                      TAG       IMAGE ID       CREATED          SIZE
infinilabs/gateway                            {{< globaldata "gateway" "version" >}}    fdae74b64e1a   47 minutes ago   23.5MB
```

## 创建配置

现在需要创建一个配置文件 `gateway.yml`，来进行基本的配置，如下：

```
path.data: data
path.logs: log

entry:
  - name: my_es_entry
    enabled: true
    router: my_router
    max_concurrency: 200000
    network:
      binding: 0.0.0.0:8000

flow:
  - name: simple_flow
    filter:
      - elasticsearch:
          elasticsearch: dev

router:
  - name: my_router
    default_flow: simple_flow

elasticsearch:
- name: dev
  enabled: true
  endpoint: http://localhost:9200
  basic_auth:
    username: test
    password: testtest
```

Note: 上面配置里面的 Elasticsearch 的相关配置，请改成实际的服务器连接地址和认证信息。

## 启动网关

使用如下命令启动极限网关容器：

```
docker run -p 2900:2900 -p 8000:8000  -v=`pwd`/gateway.yml:/gateway.yml  infinilabs/gateway:{{< globaldata "gateway" "version" >}}
```

## 验证网关

如果都运行正常的话，应该可以看到如下的信息：

```
➜  /tmp docker run -p 2900:2900 -p 8000:8000  -v=`pwd`/gateway.yml:/gateway.yml  infinilabs/gateway:{{< globaldata "gateway" "version" >}}
   ___   _   _____  __  __    __  _
  / _ \ /_\ /__   \/__\/ / /\ \ \/_\ /\_/\
 / /_\///_\\  / /\/_\  \ \/  \/ //_\\\_ _/
/ /_\\/  _  \/ / //__   \  /\  /  _  \/ \
\____/\_/ \_/\/  \__/    \/  \/\_/ \_/\_/

[GATEWAY] A light-weight, powerful and high-performance elasticsearch gateway.
[GATEWAY] 1.26.0, b61758c, Mon Dec 28 14:32:02 2024 +0800, medcl, no panic by default
[12-30 05:26:41] [INF] [instance.go:24] workspace: data/gateway/nodes/0
[12-30 05:26:41] [INF] [runner.go:59] pipeline: primary started with 1 instances
[12-30 05:26:41] [INF] [entry.go:257] entry [es_gateway] listen at: http://0.0.0.0:8000
[12-30 05:26:41] [INF] [app.go:247] gateway now started.
[12-30 05:26:45] [INF] [reverseproxy.go:196] elasticsearch [prod] endpoints: [] => [192.168.3.201:9200]
```

如果希望容器运行在后台，加上 `-d` 参数，如下：

```
docker run -d -p 2900:2900 -p 8000:8000  -v=`pwd`/gateway.yml:/gateway.yml  infinilabs/gateway:{{< globaldata "gateway" "version" >}}
```

使用命令行或者浏览器访问地址： `http://localhost:8000/` 应该就能正常访问 Elasticsearch 了，如下：

```
➜  /tmp curl -v http://localhost:8000/
*   Trying ::1...
* TCP_NODELAY set
* Connected to localhost (::1) port 8000 (#0)
> GET / HTTP/1.1
> Host: localhost:8000
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 200 OK
< Server: INFINI
< Date: Wed, 30 Dec 2020 05:12:39 GMT
< Content-Type: application/json; charset=UTF-8
< Content-Length: 480
< UPSTREAM: 192.168.3.201:9200
<
{
  "name" : "node1",
  "cluster_name" : "pi",
  "cluster_uuid" : "Z_HcN_6ESKWicV-eLsyU4g",
  "version" : {
    "number" : "6.4.2",
    "build_flavor" : "default",
    "build_type" : "tar",
    "build_hash" : "04711c2",
    "build_date" : "2018-09-26T13:34:09.098244Z",
    "build_snapshot" : false,
    "lucene_version" : "7.4.0",
    "minimum_wire_compatibility_version" : "5.6.0",
    "minimum_index_compatibility_version" : "5.0.0"
  },
  "tagline" : "You Know, for Search"
}
* Connection #0 to host localhost left intact
* Closing connection 0
```

## Docker Compose

还可以使用 Docker Compose 来管理容器实例，新建一个 `docker-compose.yml` 文件如下：

```
version: "3.5"

services:
  infini-gateway:
    image: infinilabs/gateway:{{< globaldata "gateway" "version" >}}
    ports:
      - 2900:2900
      - 8000:8000
    container_name: "infini-gateway"
    volumes:
      - ../gateway.yml:/gateway.yml

volumes:
  dist:
```

在配置文件所在目录，执行如下命令即可启动，如下：

```
➜  docker-compose up
Starting infini-gateway ... done
Attaching to infini-gateway
infini-gateway    |    ___   _   _____  __  __    __  _
infini-gateway    |   / _ \ /_\ /__   \/__\/ / /\ \ \/_\ /\_/\
infini-gateway    |  / /_\///_\\  / /\/_\  \ \/  \/ //_\\\_ _/
infini-gateway    | / /_\\/  _  \/ / //__   \  /\  /  _  \/ \
infini-gateway    | \____/\_/ \_/\/  \__/    \/  \/\_/ \_/\_/
infini-gateway    |
infini-gateway    | [GATEWAY] A light-weight, powerful and high-performance elasticsearch gateway.
infini-gateway    | [GATEWAY] 1.0.0_SNAPSHOT, b61758c, Mon Dec 28 14:32:02 2020 +0800, medcl, no panic by default
infini-gateway    | [12-30 13:24:16] [INF] [instance.go:24] workspace: data/gateway/nodes/0
infini-gateway    | [12-30 13:24:16] [INF] [api.go:244] api server listen at: http://0.0.0.0:2900
infini-gateway    | [12-30 13:24:16] [INF] [runner.go:59] pipeline: primary started with 1 instances
infini-gateway    | [12-30 13:24:16] [INF] [entry.go:257] entry [es_gateway] listen at: http://0.0.0.0:8000
infini-gateway    | [12-30 13:24:16] [INF] [app.go:247] gateway now started.
```
