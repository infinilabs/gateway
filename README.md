
INFINIBYTE, a lightweight data pipeline written in golang.

# Route

# Filter

# Output

# Features
- Auto handling upstream failure while indexing, aka nonstop indexing
- Auto detect the upstream failure in search
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

# How to use

- First, setup upstream config in the `proxy.yml`.

```
api:
  enabled: true
  network:
    binding: 0.0.0.0:2900
  tls:
    enabled: true

elasticsearch:
- name: default
  enabled: true
  endpoint: http://localhost:9200

plugins:
- name: proxy
  enabled: true
  upstream:
  - name: primary
    enabled: true
    rate_limit:
      max_qps: 10000
    queue_name: primary
    max_queue_depth: -1
    timeout: 60s
    elasticsearch: default
```
- Start the PROXY.

```
➜  elasticsearch-proxy ✗ ./bin/proxy
___  ____ ____ _  _ _   _
|__] |__/ |  |  \/   \_/
|    |  \ |__| _/\_   |
[PROXY] An elasticsearch proxy written in golang.
0.1.0_SNAPSHOT,  430bd60, Sun Apr 8 09:44:38 2018 +0800, medcl, seems good to go

[04-05 19:30:13] [INF] [instance.go:23] workspace: data/APP/nodes/0
[04-05 19:30:13] [INF] [api.go:147] api server listen at: https://0.0.0.0:2900

```

- Done! Now you are ready to rock with it.

```
➜ curl -k -XGET https://localhost:2900/
{
  "name": "PROXY",
  "tagline": "You Know, for Proxy",
  "upstream": {
    "backup": "http://localhost:9201",
    "primary": "http://localhost:9200"
  },
  "uptime": "1m58.019165s",
  "version": {
    "build_commit": "430bd60, Sun Apr 8 09:44:38 2018 +0800, medcl, seems good to go ",
    "build_date": "Sun Apr  8 09:58:29 CST 2018",
    "number": "0.1.0_SNAPSHOT"
  }
}
➜ curl -k -XGET -H'UPSTREAM:primary'  https://localhost:2900/
{
  "name" : "XZDZ8qc",
  "cluster_name" : "my-application",
  "cluster_uuid" : "FWt_UO6BRr6uBVhkVrisew",
  "version" : {
    "number" : "6.2.3",
    "build_hash" : "c59ff00",
    "build_date" : "2018-03-13T10:06:29.741383Z",
    "build_snapshot" : false,
    "lucene_version" : "7.2.1",
    "minimum_wire_compatibility_version" : "5.6.0",
    "minimum_index_compatibility_version" : "5.0.0"
  },
  "tagline" : "You Know, for Search"
}
➜ curl -k -XGET -H'UPSTREAM:backup'  https://localhost:2900/
{
  "name" : "zRcp1My",
  "cluster_name" : "elasticsearch",
  "cluster_uuid" : "FWt_UO6BRr6uBVhkVrisew",
  "version" : {
    "number" : "5.6.8",
    "build_hash" : "688ecce",
    "build_date" : "2018-02-16T16:46:30.010Z",
    "build_snapshot" : false,
    "lucene_version" : "6.6.1"
  },
  "tagline" : "You Know, for Search"
}
➜ curl -k -XPOST https://localhost:2900/myindex/_doc/1 -d'{"msg":"hello world!"}'
{ "acknowledge": true }
➜ curl -k -XGET https://localhost:2900/myindex/_doc/1
{"_index":"myindex","_type":"_doc","_id":"1","_version":1,"found":true,"_source":{"msg":"hello world!"}}
➜ curl -k -XPUT https://localhost:2900/myindex/_doc/1 -d'{"msg":"i am a proxy!"}'
{ "acknowledge": true }
➜ curl -k -XGET https://localhost:2900/myindex/_doc/1
{"_index":"myindex","_type":"_doc","_id":"1","_version":2,"found":true,"_source":{"msg":"i am a proxy!"}}
➜ curl -k -XGET https://localhost:2900/myindex/_search?q=proxy
{"took":171,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":1,"relation":"eq"},"max_score":0.8547784,"hits":[{"_index":"myindex","_type":"_doc","_id":"1","_score":0.8547784,"_source":{"msg":"i am a proxy!"}}]}}
➜ curl -k -XDELETE https://localhost:2900/myindex/_doc/1
{ "acknowledge": true }
➜ curl -k -XGET https://localhost:2900/myindex/_doc/1
{"_index":"myindex","_type":"_doc","_id":"1","found":false}
```

Have fun!

# Benchmark Test

Elasticsearch, Gateway, Load generator deployed on a single host with spec: Intel Core i5 3 GHz 6-Core, 12 GB 2400 MHz DDR4 

```
➜  http-loader git:(master) ✗ ./http-loader -c 100 -d 10 https://user:pass@localhost:8000/
1644553 requests in 9.991120172s, 1.01GB read
Requests/sec:		164601.46
Transfer/sec:		103.13MB
Avg Req Time:		607.528µs
Fastest Request:	29.71µs
Slowest Request:	76.16518ms
Number of Errors:	0
➜  http-loader git:(master) ✗ ./http-loader -c 100 -d 10 https://user:pass@localhost:8000/_search
1079734 requests in 9.993402481s, 4.45GB read
Requests/sec:		108044.68
Transfer/sec:		456.46MB
Avg Req Time:		925.543µs
Fastest Request:	37.548µs
Slowest Request:	51.972866ms
Number of Errors:	0
➜  http-loader git:(master) ✗ ./http-loader -c 100 -d 10 https://user:pass@localhost:8000/_search\?q\=message:ERROR
108297 requests in 10.183267733s, 1.12GB read
Requests/sec:		10634.80
Transfer/sec:		112.94MB
Avg Req Time:		9.403093ms
Fastest Request:	37.446µs
Slowest Request:	10.285918012s
Number of Errors:	0
➜  http-loader git:(master) ✗ ./http-loader -c 100 -d 10 https://user:pass@localhost:8000/_search\?q\=message:ERROR
968915 requests in 9.972749603s, 10.05GB read
Requests/sec:		97156.25
Transfer/sec:		1.01GB
Avg Req Time:		1.029269ms
Fastest Request:	38.208µs
Slowest Request:	230.891481ms
Number of Errors:	0
```

# Build

go1.14+

```
mkdir ~/go/src/infini.sh/ -p
cd  ~/go/src/infini.sh/
git clone https://github.com/medcl/elasticsearch-proxy.git proxy
cd proxy
make
```
Note: Path matters, please make sure follow exactly the above steps.


# Docker

The docker image size is only 8.7 MB.

Pull it from official docker hub
```
docker pull medcl/infini-gateway:latest
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

License
=======
Released under the [AGPL](https://infini.sh/LICENSE).
