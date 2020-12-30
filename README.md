
INFINI-GATEWAY, a high performance and lightweight gateway written in golang, for elasticsearch and his friends.

# Features
- Auto handling upstream failure while indexing, aka nonstop indexing
- Auto detect upstream failure in search
- Multiple write mechanism, one indexing request map to multi remote elasticsearch clusters
- Support TLS/HTTPS, generate the cert files automatically
- Support run background as daemon mode(only available on linux and macOS)
- Auto merge indexing operations to single bulk operation(WIP)
- Load balancing(indexing and search request), algorithm configurable(WIP)
- A controllable query cache layer, use redis as backend
- Index throttling or buffering, via disk based indexing queue(limit by queue length or size)
- Search throttling, limit concurrent connections to upstream(WIP)
- Builtin stats API and management UI(WIP)
- Builtin floating IP, support seamless failover and rolling upgrade
- Request logging

# Build

```
GOPATH="/Users/medcl/go" make build
```

# Benchmark

```
[root@LINUX linux64]# ./esm -s https://elastic:pass@id.domain.cn:9343 -d https://elastic:pass@id.domain.cn:8000 -x medcl2 -y medcl23 -r -w 200 --sliced_scroll_size=40 -b 5 -t=30m
medcl2
[11-12 21:05:47] [INF] [main.go:461,main] start data migration..
Scroll 20387840 / 20387840 [===================================================================================] 100.00% 1m21s
Bulk 20375408 / 20387840 [=====================================================================================]  99.94% 2m10s
[11-12 21:07:57] [INF] [main.go:492,main] data migration finished.
```

```
➜  ~ wrk   -c 1000 -d 10s -t 6 -H --latency  http://medcl:backsoon@localhost:8000
Running 10s test @ http://medcl:backsoon@localhost:8000
  6 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     4.20ms    2.06ms  59.17ms   53.33%
    Req/Sec    23.08k     5.13k   38.19k    75.50%
  1378494 requests in 10.02s, 194.57MB read
  Socket errors: connect 0, read 877, write 0, timeout 0
Requests/sec: 137595.33
Transfer/sec:     19.42MB
```


# Docker

The docker image size is only 8.7 MB.

Pull it from official docker hub
```
docker pull medcl/infini-gateway:latest
```
Or build your own image locally
```
cd ~/go/src/infini.sh/
/home/go/src/infini.sh# docker build -t medcl/infini-gateway:latest -f gateway/docker/Dockerfile .
```

Customize your `proxy.yml`, place somewhere, eg: `/tmp/proxy.yml`
```
tee /tmp/proxy.yml <<-'EOF'
elasticsearch:
- name: default
  enabled: true
  endpoint: http://192.168.3.123:9200
  index_prefix: proxy-
  basic_auth:
    username: elastic
    password: changeme
EOF
```

Rock with your proxy!
```
docker run --publish 2900:2900  -v /tmp/gateway.yml:/gateway.yml medcl/infini-gateway:latest
```

# Who are using
