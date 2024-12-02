---
weight: 35
title: Container Deployment
asciinema: true
---

# Container Deployment

INFINI Gateway supports container deployment.

## Installation Demo

{{< asciinema key="/gateway_on_docker" autoplay="1"  start-at="49" rows="30" preload="1" >}}

## Downloading an Image

The images of INFINI Gateway are published at the official repository of Docker. The URL is as follows:

[https://hub.docker.com/r/infinilabs/gateway](https://hub.docker.com/r/infinilabs/gateway)

Use the following command to obtain the latest container image:

```
docker pull infinilabs/gateway:{{< globaldata "gateway" "version" >}}
```

## Verifying the Image

After downloading the image locally, you will notice that the container image of INFINI Gateway is very small, with a size less than 25 MB. So, the downloading is very fast.

```
✗ docker images |grep "gateway" |grep "{{< globaldata "gateway" "version" >}}"
REPOSITORY                                      TAG       IMAGE ID       CREATED          SIZE
infinilabs/gateway                            {{< globaldata "gateway" "version" >}}    fdae74b64e1a   47 minutes ago   23.5MB
```

## Creating Configuration

Create a configuration file `gateway.yml` to perform basic configuration as follows:

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

Note: In the above configuration, replace the Elasticsearch configuration with the actual server connection address and authentication information.

## Starting the Gateway

Use the following command to start the INFINI Gateway container:

```
docker run -p 2900:2900 -p 8000:8000  -v=`pwd`/gateway.yml:/gateway.yml  infinilabs/gateway:{{< globaldata "gateway" "version" >}}
```

## Verifying the Gateway

If the gateway runs properly, the following information is displayed:

```
➜  /tmp docker run -p 2900:2900 -p 8000:8000  -v=`pwd`/gateway.yml:/gateway.yml  infinilabs/gateway:{{< globaldata "gateway" "version" >}}
   ___   _   _____  __  __    __  _
  / _ \ /_\ /__   \/__\/ / /\ \ \/_\ /\_/\
 / /_\///_\\  / /\/_\  \ \/  \/ //_\\\_ _/
/ /_\\/  _  \/ / //__   \  /\  /  _  \/ \
\____/\_/ \_/\/  \__/    \/  \/\_/ \_/\_/

[GATEWAY] A light-weight, powerful and high-performance elasticsearch gateway.
[GATEWAY] 1.26.0, b61758c, Mon Dec 28 14:32:02 2023 +0800, medcl, no panic by default
[12-30 05:26:41] [INF] [instance.go:24] workspace: data/gateway/nodes/0
[12-30 05:26:41] [INF] [runner.go:59] pipeline: primary started with 1 instances
[12-30 05:26:41] [INF] [entry.go:257] entry [es_gateway] listen at: http://0.0.0.0:8000
[12-30 05:26:41] [INF] [app.go:247] gateway now started.
[12-30 05:26:45] [INF] [reverseproxy.go:196] elasticsearch [prod] endpoints: [] => [192.168.3.201:9200]
```

If you want the container to run in the background, append the parameter `-d` as follows:

```
docker run -d -p 2900:2900 -p 8000:8000  -v=`pwd`/gateway.yml:/gateway.yml  infinilabs/gateway:{{< globaldata "gateway" "version" >}}
```

Access the URL `http://localhost:8000/` from the CLI or browser. The Elasticsearch can be accessed normally. See the following information.

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

You can also use docker compose to manage container instances. Create one `docker-compose.yml` file as follows:

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

In the directory where the configuration file resides, run the following command to start INFINI Gateway.

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
