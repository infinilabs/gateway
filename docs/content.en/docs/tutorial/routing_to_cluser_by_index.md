---
title: "Unified access indexes from different clusters in Kibana"
weight: 100
---

# Unified access indices from different clusters in Kibana

Now there is such a demand, customers need to divide the data according to the business dimension,
the index is split into three different clusters,
to split the large cluster into multiple small clusters have many benefits,
such as reduced coupling, bringing benefits to cluster availability and stability,
but also to avoid the impact of a single business hotspot to affect other services,
although splitting the cluster is a very common way to play, but the management is not so convenient,
especially when querying data, it may be need to access the three sets of clusters separately APIs,
even to switch between three different sets of Kibana to access the cluster's data,
is there a way to seamlessly unite them together?

## A gateway!

The answer is naturally yes, by switching the Elasticsearch address of Kibana to the address of the INFINI Gateway,
we can intelligently route requests according to the index, that is, when accessing different business indexes, they will be intelligently routed to different clusters,
as shown in the following figure:

{{% load-img "/img/smart_route_by_index.png" "" %}}

Above, we have three different indexes：

- apm-\*
- erp-\*
- mall-\*

Each corresponds to three different sets of Elasticsearch clusters:

- ES1-APM
- ES2-ERP
- ES3-MALL

Now let's see how to configure the INFINI Gateway to meet this business requirements:

## Configure clusters

First configure the connection information for the three clusters.

```
elasticsearch:
  - name: es1-apm
    enabled: true
    endpoints:
     - http://192.168.3.188:9206
  - name: es2-erp
    enabled: true
    endpoints:
     - http://192.168.3.188:9207
  - name: es3-mall
    enabled: true
    endpoints:
     - http://192.168.3.188:9208
```

## Configure Flow

then, we define three flows that are used to access three different Elasticsearch clusters, as shown below:

```
flow:
  - name: es1-flow
    filter:
      - elasticsearch:
          elasticsearch: es1-apm
  - name: es2-flow
    filter:
      - elasticsearch:
          elasticsearch: es2-erp
  - name: es3-flow
    filter:
      - elasticsearch:
          elasticsearch: es3-mall
```

Then define a flow for path rule and forwarding, as follows:

```
  - name: default-flow
    filter:
      - switch:
          remove_prefix: false
          path_rules:
            - prefix: "apm-"
              flow: es1-flow
            - prefix: "erp-"
              flow: es2-flow
            - prefix: "mall-"
              flow: es3-flow
      - flow: #default flow
          flows:
            - es1-flow
```

Match different indexes based on the index prefix in the request path and forward to different flows.

## Configure Router

Next, we define the routing information as follows:

```
router:
  - name: my_router
    default_flow: default-flow
```

Point to the default flow defined above to unify the processing of requests.

## Configure Entrypoint

Finally, we define a service that listening on port 8000 to provide unified access to Kibana, as follows:

```
entry:
  - name: es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8000
```

## Full Configuration

The final complete configuration is as follows:

```
path.data: data
path.logs: log

entry:
  - name: es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8000

flow:
  - name: default-flow
    filter:
      - switch:
          remove_prefix: false
          path_rules:
            - prefix: "apm-"
              flow: es1-flow
            - prefix: "erp-"
              flow: es2-flow
            - prefix: "mall-"
              flow: es3-flow
      - flow: #default flow
          flows:
            - es1-flow
  - name: es1-flow
    filter:
      - elasticsearch:
          elasticsearch: es1-apm
  - name: es2-flow
    filter:
      - elasticsearch:
          elasticsearch: es2-erp
  - name: es3-flow
    filter:
      - elasticsearch:
          elasticsearch: es3-mall

router:
  - name: my_router
    default_flow: default-flow

elasticsearch:
  - name: es1-apm
    enabled: true
    endpoints:
     - http://192.168.3.188:9206
  - name: es2-erp
    enabled: true
    endpoints:
     - http://192.168.3.188:9207
  - name: es3-mall
    enabled: true
    endpoints:
     - http://192.168.3.188:9208
```

## Start Gateway

Start the gateway as follows:

```
➜  gateway git:(master) ✗ ./bin/gateway -config sample-configs/elasticsearch-route-by-index.yml

   ___   _   _____  __  __    __  _
  / _ \ /_\ /__   \/__\/ / /\ \ \/_\ /\_/\
 / /_\///_\\  / /\/_\  \ \/  \/ //_\\\_ _/
/ /_\\/  _  \/ / //__   \  /\  /  _  \/ \
\____/\_/ \_/\/  \__/    \/  \/\_/ \_/\_/

[GATEWAY] A light-weight, powerful and high-performance elasticsearch gateway.
[GATEWAY] 1.0.0_SNAPSHOT, 2022-04-20 08:23:56, 2023-12-31 10:10:10, 51650a5c3d6aaa436f3c8a8828ea74894c3524b9
[04-21 13:41:21] [INF] [app.go:174] initializing gateway.
[04-21 13:41:21] [INF] [app.go:175] using config: /Users/medcl/go/src/infini.sh/gateway/sample-configs/elasticsearch-route-by-index.yml.
[04-21 13:41:21] [INF] [instance.go:72] workspace: /Users/medcl/go/src/infini.sh/gateway/data/gateway/nodes/c9bpg0ai4h931o4ngs3g
[04-21 13:41:21] [INF] [app.go:283] gateway is up and running now.
[04-21 13:41:21] [INF] [api.go:262] api listen at: http://0.0.0.0:2900
[04-21 13:41:21] [INF] [reverseproxy.go:255] elasticsearch [es1-apm] hosts: [] => [192.168.3.188:9206]
[04-21 13:41:21] [INF] [reverseproxy.go:255] elasticsearch [es2-erp] hosts: [] => [192.168.3.188:9207]
[04-21 13:41:21] [INF] [reverseproxy.go:255] elasticsearch [es3-mall] hosts: [] => [192.168.3.188:9208]
[04-21 13:41:21] [INF] [actions.go:349] elasticsearch [es2-erp] is available
[04-21 13:41:21] [INF] [actions.go:349] elasticsearch [es1-apm] is available
[04-21 13:41:21] [INF] [entry.go:312] entry [es_entry] listen at: http://0.0.0.0:8000
[04-21 13:41:21] [INF] [module.go:116] all modules are started
[04-21 13:41:21] [INF] [actions.go:349] elasticsearch [es3-mall] is available
[04-21 13:41:55] [INF] [reverseproxy.go:255] elasticsearch [es1-apm] hosts: [] => [192.168.3.188:9206]
```

After the gateway successfully started, you can access the target Elasticsearch cluster through the gateway's IP+ port 8000.

## Testing

Let's start with the API access test, as follows:

```
➜  ~ curl http://localhost:8000/apm-2022/_search -v
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 8000 (#0)
> GET /apm-2022/_search HTTP/1.1
> Host: localhost:8000
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Thu, 21 Apr 2022 05:45:44 GMT
< content-type: application/json; charset=UTF-8
< Content-Length: 162
< X-elastic-product: Elasticsearch
< X-Backend-Cluster: es1-apm
< X-Backend-Server: 192.168.3.188:9206
< X-Filters: filters->elasticsearch
<
* Connection #0 to host localhost left intact
{"took":142,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":0,"relation":"eq"},"max_score":null,"hits":[]}}%
```

You can see that `apm-2022` points to the backend `ES1-APM` cluster.

To continue testing, access to the ERP index as follows:

```
➜  ~ curl http://localhost:8000/erp-2022/_search -v
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 8000 (#0)
> GET /erp-2022/_search HTTP/1.1
> Host: localhost:8000
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Thu, 21 Apr 2022 06:24:46 GMT
< content-type: application/json; charset=UTF-8
< Content-Length: 161
< X-Backend-Cluster: es2-erp
< X-Backend-Server: 192.168.3.188:9207
< X-Filters: filters->switch->filters->elasticsearch->skipped
<
* Connection #0 to host localhost left intact
{"took":12,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":0,"relation":"eq"},"max_score":null,"hits":[]}}%
```

Great!

Let's continue testing, access to the mall index as follows:

```
➜  ~ curl http://localhost:8000/mall-2022/_search -v
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 8000 (#0)
> GET /mall-2022/_search HTTP/1.1
> Host: localhost:8000
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Thu, 21 Apr 2022 06:25:08 GMT
< content-type: application/json; charset=UTF-8
< Content-Length: 134
< X-Backend-Cluster: es3-mall
< X-Backend-Server: 192.168.3.188:9208
< X-Filters: filters->switch->filters->elasticsearch->skipped
<
* Connection #0 to host localhost left intact
{"took":8,"timed_out":false,"_shards":{"total":5,"successful":5,"skipped":0,"failed":0},"hits":{"total":0,"max_score":null,"hits":[]}}%
```

Perfect!

## Another Option

Besides using the `switch` filter, it is possible to use the path rules of the router itself, as shown in the following example configuration:

```
flow:
  - name: default_flow
    filter:
      - echo:
          message: "hello world"
  - name: mall_flow
    filter:
      - echo:
          message: "hello mall indices"
  - name: apm_flow
    filter:
      - echo:
          message: "hello apm indices"
  - name: erp_flow
    filter:
      - echo:
          message: "hello erp indices"
router:
  - name: my_router
    default_flow: default_flow
    rules:
      - method:
          - "*"
        pattern:
          - "/apm-{suffix:.*}/"
          - "/apm-{suffix:.*}/{any:.*}"
        flow:
          - apm_flow
      - method:
          - "*"
        pattern:
          - "/erp-{suffix:.*}/"
          - "/erp-{suffix:.*}/{any:.*}"
        flow:
          - erp_flow
      - method:
          - "*"
        pattern:
          - "/mall-{suffix:.*}/"
          - "/mall-{suffix:.*}/{any:.*}"
        flow:
          - mall_flow

```

INFINI Gateway has many powerful features, and there are many ways to achieve your need, go explore it by yourself.

## Modify Kibana Configuration

Modify the Kibana configuration file `kibana.yml`, replace the address of Elasticsearch with the gateway address (`HTTP: 192.168.3.200/8000`), as shown below:

```
elasticsearch.hosts: ["http://192.168.3.200:8000"]
```

Restart the Kibana。

## Visit Kibana

{{% load-img "/img/kibana-clusters-dev.jpg" "" %}}

As you can see, in the Kibana developer tool we can already perform read and write operations from three different clusters as if it were one cluster.

## Conclusion

Through the INFINI Gateway, we can be very flexible for online traffic editing, dynamic combine requests of different cluster operations together on the fly.
