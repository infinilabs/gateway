---
weight: 60
title: "vs Nginx"
draft: true
---

# INFINI Gateway Vs Nginx

## 测试说明

Nginx 是一个非常优秀的反向代理和负载均衡服务器，被广泛的用于 API 网关来负载均衡器的底座，通过和 Nginx 来进行一个横向的性能对比测试，可以帮助大家快速了解极限网关的性能情况。

### 测试场景

Gateway 和 Nginx 功能比较多，也各自有差异，我们不妨挑下面两个实际的场景来进行测试:

- 使用网关和 Nginx 来转发查询请求给后端 Elasticsearch
- 使用网关和 Nginx 来转发写入请求给后端 Elasticsearch

## 准备测试环境

| IP            | 说明                                                                                                                                                 |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| 192.168.3.199 | 压测服务器，用来执行压力测试的压力机，配置为：4C2GB                                                                                                  |
| 192.168.3.200 | 被测服务器，用来运行网关和 Nginx 的被测机，配置为：4C8GB                                                                                             |
| 192.168.3.188 | 后端服务器，Elasticsearch 服务部署在一台 Windows 服务器上，起了 4 个实例，192.168.3.188:9206\9216\9226\9236，其中（9236 为协调节点）,配置为 24C128GB |

### 服务器优化

为了充分发挥两款软件的性能，基础的调优需要提前做好，关于服务器的基础优化参考这里的设置：[Linux 系统调优](./optimization.md)

### 准备压测服务器

在 192.168.3.199 上面使用 [wrk2](https://github.com/giltene/wrk2) 作为压测工具来进行压测：

首先安装依赖的系统库：

```
apt-get install libcurl4-gnutls-dev libexpat1-dev gettext \
  libz-dev libssl-dev
```

下载压测工具源代码并进行本地编译：

```
cd /opt/
git clone https://github.com/giltene/wrk2.git
cd wrk2/
make
mv wrk  /usr/bin/
```

### 准备被测服务器

在 192.168.3.200 上面，分别安装 Nginx 和 Gateway，确保他们都使用相同规格的系统和硬件资源。

安装 Nginx

```
apt install nginx-full

```

优化 Nginx 的配置，发挥其最大性能（如有更佳优化参数，请联系我们）：

```
vi /etc/nginx/nginx.conf
worker_processes auto;
events {
        worker_connections 100000;
        # multi_accept on;
}
http {
        sendfile on;
        tcp_nopush on;
        tcp_nodelay on;
        keepalive_timeout 65;
        types_hash_max_size 2048;
...
```

## 查询压力测试

### 测试代理单节点

首先，我们测试一下代理单个后端 Elasticsearch 的性能情况。

新增一个 Nginx 的配置，用来代理请求并转发给后端的 Elasticsearch 协调节点，如下：

```
root@gateway:/tmp# cat /etc/nginx/conf.d/es.conf
server {
    listen 0.0.0.0:9090;
    access_log off;

location / {
    proxy_pass http://192.168.3.188:9236;
}
}
```

新增一个 Gateway 的配置，实现同样的转发需求：

```
root@gateway:/tmp# cat sample-configs/elasticsearch-proxy.yml
path.data: data
path.logs: log

entry:
  - name: my_es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8000

flow:
  - name: es-flow
    filter:
      - elasticsearch:
          elasticsearch: es-server

router:
  - name: my_router
    default_flow: es-flow

elasticsearch:
  - name: es-server
    enabled: true
    hosts:
     - 192.168.3.188:9236
```

启动网关和重启 Nginx：

```
systemctl restart nginx
./gateway-linux-amd64 -config sample-configs/elasticsearch-proxy.yml
```

现在 Nginx 应该监听了 `9090` 端口，Gateway 监听了 `8000`端口。

#### 准备 ES

在 Elasticsearch 里面新建一个索引，并插入一条简单的记录，用于后续的检索测试：

```
POST test/_doc/1
{
  "tag":"hello world"
}
GET test/_search
{
  "took": 162,
  "timed_out": false,
  "_shards": {
    "total": 1,
    "successful": 1,
    "skipped": 0,
    "failed": 0
  },
  "hits": {
    "total": {
      "value": 1,
      "relation": "eq"
    },
    "max_score": 1,
    "hits": [
      {
        "_index": "test",
        "_type": "_doc",
        "_id": "1",
        "_score": 1,
        "_source": {
          "tag": "hello world"
        }
      }
    ]
  }
}
```

#### 执行测试

通过访问不同的端口就可以访问不同的代理后端，分别如下：

- 访问 Nginx：http://192.168.3.200:9090/test/_search
- 访问 Gateway：http://192.168.3.200:8000/test/_search

压测命令如下：

```
wrk -t10 -c1000 -d30s -R 100000   http://目标地址/test/_search
```

使用相同的参数分别执行压测，可以看到压测端和被测端的 CPU 都已经跑满 100%，说明已经最大化利用了当前服务器资源。

{{% load-img "/img/nginx_vs_gateway_nginx_at_c1000_system_load.png" "" %}}
{{% load-img "/img/nginx_vs_gateway_gateway_at_c1000_system_load.png" "" %}}

Elasticsearch 的多个节点实际部署在一台物理机上面：
{{% load-img "/img/nginx_vs_gateway_elasticsearch-cpu.jpg" "" %}}

可以看到后端的 Elasticsearch 资源充足，CPU 都没有跑满，Elasticsearch 端无瓶颈。

压测结果最终输出如下：

{{< expand "展开查看详细压测结果" "..." >}}

```
root@loadgen:/opt/wrk2# wrk -t10 -c1000 -d30s -R 100000   http://192.168.3.200:9090/test/_search
Running 30s test @ http://192.168.3.200:9090/test/_search
  10 threads and 1000 connections
  Thread calibration: mean lat.: 4016.040ms, rate sampling interval: 15695ms
  Thread calibration: mean lat.: 4932.029ms, rate sampling interval: 16449ms
  Thread calibration: mean lat.: 4901.524ms, rate sampling interval: 15974ms
  Thread calibration: mean lat.: 4812.219ms, rate sampling interval: 15941ms
  Thread calibration: mean lat.: 4065.803ms, rate sampling interval: 15392ms
  Thread calibration: mean lat.: 4781.841ms, rate sampling interval: 15990ms
  Thread calibration: mean lat.: 4875.765ms, rate sampling interval: 15917ms
  Thread calibration: mean lat.: 4820.694ms, rate sampling interval: 16392ms
  Thread calibration: mean lat.: 4769.239ms, rate sampling interval: 15785ms
  Thread calibration: mean lat.: 2100.813ms, rate sampling interval: 13082ms
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    17.91s     5.53s   28.93s    56.88%
    Req/Sec   502.90     45.96   553.00     80.00%
  153096 requests in 30.01s, 67.74MB read
  Socket errors: connect 0, read 0, write 0, timeout 2136
Requests/sec:   5101.89
Transfer/sec:      2.26MB
root@loadgen:/opt/wrk2# wrk -t10 -c1000 -d30s -R 100000   http://192.168.3.200:800/test/_search
unable to connect to 192.168.3.200:800 Connection refused
root@loadgen:/opt/wrk2# ^C
root@loadgen:/opt/wrk2# wrk -t10 -c1000 -d30s -R 100000   http://192.168.3.200:8000/test/_search
Running 30s test @ http://192.168.3.200:8000/test/_search
  10 threads and 1000 connections
  Thread calibration: mean lat.: 3118.678ms, rate sampling interval: 11198ms
  Thread calibration: mean lat.: 2652.429ms, rate sampling interval: 9936ms
  Thread calibration: mean lat.: 3010.793ms, rate sampling interval: 10674ms
  Thread calibration: mean lat.: 3155.238ms, rate sampling interval: 11313ms
  Thread calibration: mean lat.: 2962.527ms, rate sampling interval: 10665ms
  Thread calibration: mean lat.: 3061.773ms, rate sampling interval: 10797ms
  Thread calibration: mean lat.: 3130.273ms, rate sampling interval: 11165ms
  Thread calibration: mean lat.: 2765.918ms, rate sampling interval: 10100ms
  Thread calibration: mean lat.: 3080.733ms, rate sampling interval: 10797ms
  Thread calibration: mean lat.: 2301.280ms, rate sampling interval: 9953ms
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    12.18s     3.68s   19.55s    57.90%
    Req/Sec     1.79k   185.76     2.04k    60.00%
  533703 requests in 29.99s, 260.62MB read
  Socket errors: connect 0, read 0, write 0, timeout 6467
Requests/sec:  17795.49
Transfer/sec:      8.69MB
```

{{< /expand >}}

#### 测试结果

可以看到 wrk 走 Nginx 的 qps 吞吐为 ~5100/s，而走 Gateway 的吞吐则为 ~17000/s，差距悬殊，Gateway 的吞吐为 Nginx 的 3 倍。

查看 Elasticsearch 端监控走势如下图：

{{% load-img "/img/nginx_vs_gateway_search_at_c1000.jpg" "" %}}

> Tips：这里是用的 INFINI Labs 自家开发的[监控管理工具](http://console.infinilabs.com/)，用了都说好。

基本上和 wrk 的结果一致呼应，走 Gateway 要比走 Nginx 查询吞吐高不少。

为了避免后端查询缓存的影响，可以多测几遍：
{{< expand "展开查看详细压测结果" "..." >}}

```
root@loadgen:/opt/wrk2# wrk -t10 -c1000 -d30s -R 100000   http://192.168.3.200:9090/test/_search
Running 30s test @ http://192.168.3.200:9090/test/_search
  10 threads and 1000 connections
  Thread calibration: mean lat.: 3737.755ms, rate sampling interval: 15884ms
  Thread calibration: mean lat.: 4808.373ms, rate sampling interval: 16031ms
  Thread calibration: mean lat.: 4939.159ms, rate sampling interval: 16687ms
  Thread calibration: mean lat.: 5071.944ms, rate sampling interval: 16334ms
  Thread calibration: mean lat.: 4860.762ms, rate sampling interval: 16343ms
  Thread calibration: mean lat.: 5245.414ms, rate sampling interval: 16941ms
  Thread calibration: mean lat.: 4620.354ms, rate sampling interval: 15794ms
  Thread calibration: mean lat.: 4830.987ms, rate sampling interval: 15949ms
  Thread calibration: mean lat.: 2416.157ms, rate sampling interval: 14147ms
  Thread calibration: mean lat.: 3931.905ms, rate sampling interval: 15302ms
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    18.41s     5.35s   28.43s    59.01%
    Req/Sec   423.30     41.97   495.00     60.00%
  128952 requests in 29.99s, 57.06MB read
  Socket errors: connect 0, read 87, write 0, timeout 2312
Requests/sec:   4299.57
Transfer/sec:      1.90MB
root@loadgen:/opt/wrk2# wrk -t10 -c1000 -d30s -R 100000   http://192.168.3.200:8000/test/_search
Running 30s test @ http://192.168.3.200:8000/test/_search
  10 threads and 1000 connections
  Thread calibration: mean lat.: 3050.983ms, rate sampling interval: 11231ms
  Thread calibration: mean lat.: 2338.112ms, rate sampling interval: 10076ms
  Thread calibration: mean lat.: 3155.022ms, rate sampling interval: 11231ms
  Thread calibration: mean lat.: 3017.262ms, rate sampling interval: 10452ms
  Thread calibration: mean lat.: 2336.867ms, rate sampling interval: 10035ms
  Thread calibration: mean lat.: 3062.089ms, rate sampling interval: 10862ms
  Thread calibration: mean lat.: 3005.530ms, rate sampling interval: 10461ms
  Thread calibration: mean lat.: 2753.639ms, rate sampling interval: 10018ms
  Thread calibration: mean lat.: 2299.394ms, rate sampling interval: 9928ms
  Thread calibration: mean lat.: 2884.390ms, rate sampling interval: 10715ms
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    11.85s     3.56s   19.10s    57.90%
    Req/Sec     1.81k   187.28     2.14k    60.00%
  530550 requests in 30.00s, 259.08MB read
  Socket errors: connect 0, read 0, write 0, timeout 7008
Requests/sec:  17684.97
Transfer/sec:      8.64MB
```

{{< /expand >}}

调整测试顺序再来一遍：

{{< expand "展开查看详细压测结果" "..." >}}

```
root@loadgen:/opt/wrk2# wrk -t10 -c1000 -d30s -R 100000   http://192.168.3.200:8000/test/_search
Running 30s test @ http://192.168.3.200:8000/test/_search
  10 threads and 1000 connections
  Thread calibration: mean lat.: 2548.693ms, rate sampling interval: 10199ms
  Thread calibration: mean lat.: 2668.036ms, rate sampling interval: 10674ms
  Thread calibration: mean lat.: 2877.709ms, rate sampling interval: 10035ms
  Thread calibration: mean lat.: 2829.798ms, rate sampling interval: 10436ms
  Thread calibration: mean lat.: 3100.091ms, rate sampling interval: 11091ms
  Thread calibration: mean lat.: 2623.475ms, rate sampling interval: 10248ms
  Thread calibration: mean lat.: 2865.326ms, rate sampling interval: 10362ms
  Thread calibration: mean lat.: 2883.781ms, rate sampling interval: 10608ms
  Thread calibration: mean lat.: 3116.084ms, rate sampling interval: 10862ms
  Thread calibration: mean lat.: 3065.616ms, rate sampling interval: 11034ms
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    12.05s     3.55s   19.17s    58.76%
    Req/Sec     1.85k   153.99     2.04k    70.00%
  543553 requests in 29.99s, 265.43MB read
  Socket errors: connect 0, read 0, write 0, timeout 6748
Requests/sec:  18121.52
Transfer/sec:      8.85MB
root@loadgen:/opt/wrk2# wrk -t10 -c1000 -d30s -R 100000   http://192.168.3.200:9090/test/_search
Running 30s test @ http://192.168.3.200:9090/test/_search
  10 threads and 1000 connections
  Thread calibration: mean lat.: 4260.723ms, rate sampling interval: 17563ms
  Thread calibration: mean lat.: 4012.010ms, rate sampling interval: 16080ms
  Thread calibration: mean lat.: 4125.644ms, rate sampling interval: 17399ms
  Thread calibration: mean lat.: 4867.496ms, rate sampling interval: 16171ms
  Thread calibration: mean lat.: 3725.344ms, rate sampling interval: 16285ms
  Thread calibration: mean lat.: 5318.414ms, rate sampling interval: 16605ms
  Thread calibration: mean lat.: 4411.084ms, rate sampling interval: 17317ms
  Thread calibration: mean lat.: 5322.922ms, rate sampling interval: 17285ms
  Thread calibration: mean lat.: 6209.682ms, rate sampling interval: 17530ms
  Thread calibration: mean lat.: 3973.158ms, rate sampling interval: 16367ms
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    18.68s     5.37s   29.07s    58.59%
    Req/Sec   438.00     24.01   482.00     70.00%
  130708 requests in 30.00s, 57.84MB read
  Socket errors: connect 0, read 103, write 0, timeout 2129
Requests/sec:   4356.26
Transfer/sec:      1.93MB
```

{{< /expand >}}

多次重复测试结果没有产生变化，测试结果稳定，Gateway 完胜 Nginx。

### 测试代理多节点

这次我们通过在 Nginx 和网关里面配置多个 Elasticsearch 后端节点信息，充分利用后端资源来进行转发。

#### 相关配置

{{< expand "展开查看详细配置信息" "..." >}}

```
root@gateway:/etc/nginx/conf.d# cat es.conf
upstream servers {
    server 192.168.3.188:9236;
    server 192.168.3.188:9226;
    server 192.168.3.188:9216;
    server 192.168.3.188:9206;
    keepalive 300;
  }

server {
    listen 0.0.0.0:9090;
    access_log off;

location / {
    proxy_pass http://servers;
}
}

root@gateway:/tmp# cat sample-configs/elasticsearch-proxy.yml
path.data: data
path.logs: log

stats:
  enabled: false

entry:
  - name: my_es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8000

flow:
  - name: es-flow
    filter:
      - elasticsearch:
          elasticsearch: es-server

router:
  - name: my_router
    default_flow: es-flow

elasticsearch:
  - name: es-server
    enabled: true
    hosts:
     - 192.168.3.188:9236
     - 192.168.3.188:9226
     - 192.168.3.188:9216
     - 192.168.3.188:9206
```

{{< /expand >}}

#### 执行测试

测试结果输出：
{{< expand "展开查看详细压测结果" "..." >}}

```
root@loadgen:/opt/wrk2# wrk -t10 -c1000 -d30s -R 100000   http://192.168.3.200:9090/test/_search
Running 30s test @ http://192.168.3.200:9090/test/_search
  10 threads and 1000 connections
  Thread calibration: mean lat.: 3632.311ms, rate sampling interval: 15745ms
  Thread calibration: mean lat.: 4274.681ms, rate sampling interval: 15663ms
  Thread calibration: mean lat.: 3119.798ms, rate sampling interval: 14188ms
  Thread calibration: mean lat.: 4476.679ms, rate sampling interval: 15622ms
  Thread calibration: mean lat.: 4511.473ms, rate sampling interval: 15966ms
  Thread calibration: mean lat.: 4530.511ms, rate sampling interval: 16891ms
  Thread calibration: mean lat.: 5506.308ms, rate sampling interval: 17268ms
  Thread calibration: mean lat.: 4368.631ms, rate sampling interval: 15368ms
  Thread calibration: mean lat.: 4219.582ms, rate sampling interval: 15155ms
  Thread calibration: mean lat.: 5381.712ms, rate sampling interval: 17104ms
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    17.94s     5.42s   28.67s    57.15%
    Req/Sec   505.90     28.54   553.00     60.00%
  152832 requests in 29.99s, 67.63MB read
  Socket errors: connect 0, read 0, write 0, timeout 2560
Requests/sec:   5095.85
Transfer/sec:      2.25MB
root@loadgen:/opt/wrk2# wrk -t10 -c1000 -d30s -R 100000   http://192.168.3.200:8000/test/_search
Running 30s test @ http://192.168.3.200:8000/test/_search
  10 threads and 1000 connections
  Thread calibration: mean lat.: 3013.227ms, rate sampling interval: 11042ms
  Thread calibration: mean lat.: 2298.560ms, rate sampling interval: 10846ms
  Thread calibration: mean lat.: 2988.278ms, rate sampling interval: 10821ms
  Thread calibration: mean lat.: 3157.630ms, rate sampling interval: 10960ms
  Thread calibration: mean lat.: 2309.700ms, rate sampling interval: 10141ms
  Thread calibration: mean lat.: 3121.468ms, rate sampling interval: 10756ms
  Thread calibration: mean lat.: 2852.655ms, rate sampling interval: 10665ms
  Thread calibration: mean lat.: 3298.113ms, rate sampling interval: 11575ms
  Thread calibration: mean lat.: 2885.254ms, rate sampling interval: 10657ms
  Thread calibration: mean lat.: 3112.382ms, rate sampling interval: 11231ms
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    12.25s     3.68s   19.63s    57.56%
    Req/Sec     1.75k   201.70     1.99k    70.00%
  518703 requests in 30.00s, 253.31MB read
  Socket errors: connect 0, read 0, write 0, timeout 6679
Requests/sec:  17291.00
Transfer/sec:      8.44MB
```

{{< /expand >}}

由于瓶颈都在代理端（CPU 资源已跑满），实测增加多个后端代理节点确实没有带来太大的差异。

#### 测试结果

从请求转发的处理吞吐能力来看，Gateway 完胜 Nginx。

### 测试延迟

鉴于 Nginx 的转发能力提升不上去，我们接下来测试延迟，将压力控制在 3000 并发，来比较他们的延迟情况。

测试时间调长到 60s，吞吐限制控制到 3000，并使用 wrk 来记录客户端请求的延迟信息。

#### 执行测试

首先测试 Nginx，运行结果如下：

{{< expand "展开查看详细压测结果" "..." >}}

```
root@loadgen:/opt/wrk2# wrk -t10 -c1000 -d60s -R 3000 --u_latency  http://192.168.3.200:9090/test/_search
Running 1m test @ http://192.168.3.200:9090/test/_search
  10 threads and 1000 connections
  Thread calibration: mean lat.: 33.582ms, rate sampling interval: 102ms
  Thread calibration: mean lat.: 220.602ms, rate sampling interval: 1516ms
  Thread calibration: mean lat.: 188.658ms, rate sampling interval: 1460ms
  Thread calibration: mean lat.: 208.472ms, rate sampling interval: 1511ms
  Thread calibration: mean lat.: 182.986ms, rate sampling interval: 1434ms
  Thread calibration: mean lat.: 616.319ms, rate sampling interval: 4259ms
  Thread calibration: mean lat.: 150.266ms, rate sampling interval: 847ms
  Thread calibration: mean lat.: 185.063ms, rate sampling interval: 1423ms
  Thread calibration: mean lat.: 156.954ms, rate sampling interval: 1043ms
  Thread calibration: mean lat.: 170.819ms, rate sampling interval: 1111ms
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     3.58s     4.67s   22.97s    82.90%
    Req/Sec   237.67    221.67     0.99k    76.14%
  Latency Distribution (HdrHistogram - Recorded Latency)
 50.000%    1.45s
 75.000%    5.46s
 90.000%   10.75s
 99.000%   18.55s
 99.900%   20.89s
 99.990%   22.20s
 99.999%   22.76s
100.000%   22.99s

  Detailed Percentile spectrum:
       Value   Percentile   TotalCount 1/(1-Percentile)

       1.458     0.000000            1         1.00
      14.791     0.100000        10606         1.11
      46.335     0.200000        21212         1.25
     236.543     0.300000        31820         1.43
     575.999     0.400000        42419         1.67
    1446.911     0.500000        53035         2.00
    1930.239     0.550000        58328         2.22
    2596.863     0.600000        63631         2.50
    3211.263     0.650000        68953         2.86
    4218.879     0.700000        74236         3.33
    5459.967     0.750000        79541         4.00
    6189.055     0.775000        82192         4.44
    7118.847     0.800000        84859         5.00
    8122.367     0.825000        87492         5.71
    9117.695     0.850000        90156         6.67
    9994.239     0.875000        92797         8.00
   10403.839     0.887500        94135         8.89
   10747.903     0.900000        95464        10.00
   11247.615     0.912500        96775        11.43
   12025.855     0.925000        98103        13.33
   12967.935     0.937500        99421        16.00
   13402.111     0.943750       100092        17.78
   13910.015     0.950000       100747        20.00
   14524.415     0.956250       101406        22.86
   15187.967     0.962500       102079        26.67
   15835.135     0.968750       102732        32.00
   16121.855     0.971875       103063        35.56
   16433.151     0.975000       103399        40.00
   16826.367     0.978125       103729        45.71
   17334.271     0.981250       104061        53.33
   17825.791     0.984375       104395        64.00
   18006.015     0.985938       104556        71.11
   18219.007     0.987500       104729        80.00
   18399.231     0.989062       104887        91.43
   18644.991     0.990625       105054       106.67
   18890.751     0.992188       105218       128.00
   19038.207     0.992969       105301       142.22
   19185.663     0.993750       105386       160.00
   19316.735     0.994531       105466       182.86
   19464.191     0.995313       105549       213.33
   19644.415     0.996094       105631       256.00
   19726.335     0.996484       105673       284.44
   19873.791     0.996875       105717       320.00
   20021.247     0.997266       105759       365.71
   20185.087     0.997656       105800       426.67
   20365.311     0.998047       105840       512.00
   20463.615     0.998242       105860       568.89
   20561.919     0.998437       105881       640.00
   20692.991     0.998633       105903       731.43
   20791.295     0.998828       105921       853.33
   20905.983     0.999023       105945      1024.00
   20971.519     0.999121       105952      1137.78
   21053.439     0.999219       105963      1280.00
   21151.743     0.999316       105973      1462.86
   21282.815     0.999414       105984      1706.67
   21463.039     0.999512       105994      2048.00
   21528.575     0.999561       106000      2275.56
   21577.727     0.999609       106004      2560.00
   21676.031     0.999658       106009      2925.71
   21741.567     0.999707       106014      3413.33
   21823.487     0.999756       106020      4096.00
   21889.023     0.999780       106022      4551.11
   21921.791     0.999805       106025      5120.00
   22003.711     0.999829       106028      5851.43
   22020.095     0.999854       106030      6826.67
   22167.551     0.999878       106033      8192.00
   22200.319     0.999890       106034      9102.22
   22216.703     0.999902       106035     10240.00
   22233.087     0.999915       106036     11702.86
   22249.471     0.999927       106038     13653.33
   22265.855     0.999939       106039     16384.00
   22298.623     0.999945       106040     18204.44
   22298.623     0.999951       106040     20480.00
   22331.391     0.999957       106041     23405.71
   22495.231     0.999963       106042     27306.67
   22495.231     0.999969       106042     32768.00
   22511.615     0.999973       106043     36408.89
   22511.615     0.999976       106043     40960.00
   22511.615     0.999979       106043     46811.43
   22757.375     0.999982       106044     54613.33
   22757.375     0.999985       106044     65536.00
   22757.375     0.999986       106044     72817.78
   22757.375     0.999988       106044     81920.00
   22757.375     0.999989       106044     93622.86
   22986.751     0.999991       106045    109226.67
   22986.751     1.000000       106045          inf
#[Mean    =     3580.352, StdDeviation   =     4671.049]
#[Max     =    22970.368, Total count    =       106045]
#[Buckets =           27, SubBuckets     =         2048]
----------------------------------------------------------

  Latency Distribution (HdrHistogram - Uncorrected Latency (measured without taking delayed starts into account))
 50.000%  293.38ms
 75.000%  499.97ms
 90.000%  662.53ms
 99.000%    1.10s
 99.900%    7.13s
 99.990%    7.70s
 99.999%    7.93s
100.000%    7.94s

  Detailed Percentile spectrum:
       Value   Percentile   TotalCount 1/(1-Percentile)

       1.329     0.000000            1         1.00
       9.399     0.100000        10605         1.11
      28.495     0.200000        21219         1.25
      59.487     0.300000        31817         1.43
     207.871     0.400000        42428         1.67
     293.375     0.500000        53045         2.00
     332.543     0.550000        58326         2.22
     389.887     0.600000        63633         2.50
     422.399     0.650000        68934         2.86
     462.847     0.700000        74238         3.33
     499.967     0.750000        79573         4.00
     523.263     0.775000        82206         4.44
     551.935     0.800000        84854         5.00
     576.511     0.825000        87528         5.71
     599.551     0.850000        90169         6.67
     629.247     0.875000        92841         8.00
     645.631     0.887500        94154         8.89
     662.527     0.900000        95462        10.00
     686.079     0.912500        96774        11.43
     733.695     0.925000        98115        13.33
     767.999     0.937500        99436        16.00
     782.847     0.943750       100082        17.78
     796.671     0.950000       100746        20.00
     809.983     0.956250       101418        22.86
     829.439     0.962500       102070        26.67
     865.791     0.968750       102735        32.00
     892.927     0.971875       103063        35.56
    1023.999     0.975000       103400        40.00
    1041.919     0.978125       103731        45.71
    1054.719     0.981250       104083        53.33
    1064.959     0.984375       104419        64.00
    1069.055     0.985938       104580        71.11
    1076.223     0.987500       104738        80.00
    1089.535     0.989062       104892        91.43
    1112.063     0.990625       105055       106.67
    1166.335     0.992188       105218       128.00
    1419.263     0.992969       105300       142.22
    1842.175     0.993750       105383       160.00
    2381.823     0.994531       105466       182.86
    3043.327     0.995313       105548       213.33
    3071.999     0.996094       105634       256.00
    3207.167     0.996484       105674       284.44
    3293.183     0.996875       105714       320.00
    3672.063     0.997266       105756       365.71
    4853.759     0.997656       105797       426.67
    5779.455     0.998047       105838       512.00
    6582.271     0.998242       105859       568.89
    7081.983     0.998437       105881       640.00
    7118.847     0.998633       105904       731.43
    7127.039     0.998828       105934       853.33
    7131.135     0.999023       105960      1024.00
    7131.135     0.999121       105960      1137.78
    7135.231     0.999219       105981      1280.00
    7135.231     0.999316       105981      1462.86
    7139.327     0.999414       105984      1706.67
    7311.359     0.999512       105994      2048.00
    7397.375     0.999561       105999      2275.56
    7434.239     0.999609       106004      2560.00
    7487.487     0.999658       106009      2925.71
    7528.447     0.999707       106015      3413.33
    7598.079     0.999756       106020      4096.00
    7626.751     0.999780       106022      4551.11
    7643.135     0.999805       106025      5120.00
    7675.903     0.999829       106027      5851.43
    7688.191     0.999854       106031      6826.67
    7696.383     0.999878       106033      8192.00
    7700.479     0.999890       106034      9102.22
    7708.671     0.999902       106035     10240.00
    7778.303     0.999915       106036     11702.86
    7790.591     0.999927       106038     13653.33
    7819.263     0.999939       106039     16384.00
    7835.647     0.999945       106040     18204.44
    7835.647     0.999951       106040     20480.00
    7843.839     0.999957       106041     23405.71
    7852.031     0.999963       106042     27306.67
    7852.031     0.999969       106042     32768.00
    7868.415     0.999973       106043     36408.89
    7868.415     0.999976       106043     40960.00
    7868.415     0.999979       106043     46811.43
    7929.855     0.999982       106044     54613.33
    7929.855     0.999985       106044     65536.00
    7929.855     0.999986       106044     72817.78
    7929.855     0.999988       106044     81920.00
    7929.855     0.999989       106044     93622.86
    7942.143     0.999991       106045    109226.67
    7942.143     1.000000       106045          inf
#[Mean    =      338.201, StdDeviation   =      447.413]
#[Max     =     7938.048, Total count    =       106045]
#[Buckets =           27, SubBuckets     =         2048]
----------------------------------------------------------
  133647 requests in 1.00m, 59.14MB read
  Socket errors: connect 0, read 78, write 0, timeout 3452
Requests/sec:   2226.03
Transfer/sec:      0.99MB
```

{{< /expand >}}

接下来再看 Gateway 的测试结果：
{{< expand "展开查看详细压测结果" "..." >}}

```
root@loadgen:/opt/wrk2# wrk -t10 -c1000 -d60s -R 3000 --u_latency  http://192.168.3.200:8000/test/_search
Running 1m test @ http://192.168.3.200:8000/test/_search
  10 threads and 1000 connections
  Thread calibration: mean lat.: 7.919ms, rate sampling interval: 29ms
  Thread calibration: mean lat.: 17.213ms, rate sampling interval: 38ms
  Thread calibration: mean lat.: 19.008ms, rate sampling interval: 73ms
  Thread calibration: mean lat.: 21.199ms, rate sampling interval: 75ms
  Thread calibration: mean lat.: 9.096ms, rate sampling interval: 29ms
  Thread calibration: mean lat.: 16.224ms, rate sampling interval: 52ms
  Thread calibration: mean lat.: 11.586ms, rate sampling interval: 33ms
  Thread calibration: mean lat.: 20.279ms, rate sampling interval: 73ms
  Thread calibration: mean lat.: 15.045ms, rate sampling interval: 47ms
  Thread calibration: mean lat.: 16.823ms, rate sampling interval: 42ms
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    12.87ms   16.58ms 243.58ms   94.46%
    Req/Sec   151.30    291.03     1.50k    88.84%
  Latency Distribution (HdrHistogram - Recorded Latency)
 50.000%    9.62ms
 75.000%   15.16ms
 90.000%   19.85ms
 99.000%   94.14ms
 99.900%  170.24ms
 99.990%  201.34ms
 99.999%  219.39ms
100.000%  243.71ms

  Detailed Percentile spectrum:
       Value   Percentile   TotalCount 1/(1-Percentile)

       0.919     0.000000            1         1.00
       2.913     0.100000         7236         1.11
       3.825     0.200000        14460         1.25
       5.023     0.300000        21694         1.43
       6.919     0.400000        28916         1.67
       9.623     0.500000        36140         2.00
      10.799     0.550000        39752         2.22
      12.295     0.600000        43368         2.50
      13.399     0.650000        46995         2.86
      14.247     0.700000        50617         3.33
      15.159     0.750000        54203         4.00
      15.639     0.775000        56025         4.44
      16.295     0.800000        57827         5.00
      17.087     0.825000        59624         5.71
      17.887     0.850000        61421         6.67
      18.783     0.875000        63234         8.00
      19.359     0.887500        64137         8.89
      19.855     0.900000        65040        10.00
      20.735     0.912500        65941        11.43
      22.111     0.925000        66841        13.33
      25.935     0.937500        67743        16.00
      28.911     0.943750        68196        17.78
      32.831     0.950000        68647        20.00
      36.447     0.956250        69098        22.86
      44.767     0.962500        69549        26.67
      51.039     0.968750        70001        32.00
      55.103     0.971875        70226        35.56
      59.839     0.975000        70452        40.00
      66.239     0.978125        70682        45.71
      73.983     0.981250        70905        53.33
      78.271     0.984375        71130        64.00
      83.007     0.985938        71245        71.11
      84.927     0.987500        71356        80.00
      91.967     0.989062        71469        91.43
      95.615     0.990625        71582       106.67
     102.591     0.992188        71696       128.00
     105.471     0.992969        71752       142.22
     108.735     0.993750        71808       160.00
     115.903     0.994531        71863       182.86
     123.967     0.995313        71921       213.33
     133.247     0.996094        71976       256.00
     139.903     0.996484        72006       284.44
     144.383     0.996875        72036       320.00
     147.455     0.997266        72063       365.71
     148.863     0.997656        72089       426.67
     157.951     0.998047        72118       512.00
     158.463     0.998242        72134       568.89
     162.559     0.998437        72146       640.00
     164.351     0.998633        72160       731.43
     167.679     0.998828        72174       853.33
     170.239     0.999023        72189      1024.00
     170.495     0.999121        72196      1137.78
     174.463     0.999219        72202      1280.00
     178.559     0.999316        72209      1462.86
     182.399     0.999414        72216      1706.67
     187.007     0.999512        72223      2048.00
     187.519     0.999561        72229      2275.56
     187.647     0.999609        72230      2560.00
     188.671     0.999658        72234      2925.71
     190.079     0.999707        72237      3413.33
     195.711     0.999756        72241      4096.00
     196.095     0.999780        72243      4551.11
     196.223     0.999805        72246      5120.00
     196.223     0.999829        72246      5851.43
     199.039     0.999854        72248      6826.67
     201.215     0.999878        72250      8192.00
     201.343     0.999890        72251      9102.22
     201.343     0.999902        72251     10240.00
     203.263     0.999915        72252     11702.86
     204.031     0.999927        72253     13653.33
     211.455     0.999939        72255     16384.00
     211.455     0.999945        72255     18204.44
     211.455     0.999951        72255     20480.00
     211.455     0.999957        72255     23405.71
     219.135     0.999963        72256     27306.67
     219.135     0.999969        72256     32768.00
     219.391     0.999973        72257     36408.89
     219.391     0.999976        72257     40960.00
     219.391     0.999979        72257     46811.43
     219.391     0.999982        72257     54613.33
     219.391     0.999985        72257     65536.00
     243.711     0.999986        72258     72817.78
     243.711     1.000000        72258          inf
#[Mean    =       12.868, StdDeviation   =       16.584]
#[Max     =      243.584, Total count    =        72258]
#[Buckets =           27, SubBuckets     =         2048]
----------------------------------------------------------

  Latency Distribution (HdrHistogram - Uncorrected Latency (measured without taking delayed starts into account))
 50.000%    3.72ms
 75.000%    5.67ms
 90.000%   17.55ms
 99.000%   82.69ms
 99.900%  162.82ms
 99.990%  197.76ms
 99.999%  219.01ms
100.000%  228.10ms

  Detailed Percentile spectrum:
       Value   Percentile   TotalCount 1/(1-Percentile)

       0.786     0.000000            1         1.00
       1.791     0.100000         7230         1.11
       2.381     0.200000        14482         1.25
       2.863     0.300000        21704         1.43
       3.327     0.400000        28930         1.67
       3.719     0.500000        36162         2.00
       3.929     0.550000        39753         2.22
       4.187     0.600000        43376         2.50
       4.539     0.650000        47010         2.86
       4.987     0.700000        50595         3.33
       5.671     0.750000        54212         4.00
       6.227     0.775000        56005         4.44
       7.307     0.800000        57809         5.00
      12.855     0.825000        59670         5.71
      13.855     0.850000        61464         6.67
      15.599     0.875000        63226         8.00
      16.799     0.887500        64159         8.89
      17.551     0.900000        65033        10.00
      18.351     0.912500        65937        11.43
      18.975     0.925000        66849        13.33
      22.047     0.937500        67743        16.00
      23.903     0.943750        68198        17.78
      25.759     0.950000        68646        20.00
      29.935     0.956250        69098        22.86
      34.879     0.962500        69550        26.67
      44.383     0.968750        70001        32.00
      47.903     0.971875        70233        35.56
      51.839     0.975000        70452        40.00
      57.919     0.978125        70679        45.71
      64.959     0.981250        70905        53.33
      71.807     0.984375        71131        64.00
      75.135     0.985938        71243        71.11
      78.015     0.987500        71355        80.00
      81.919     0.989062        71472        91.43
      84.095     0.990625        71585       106.67
      91.839     0.992188        71695       128.00
      95.743     0.992969        71752       142.22
     102.591     0.993750        71807       160.00
     109.375     0.994531        71863       182.86
     119.167     0.995313        71922       213.33
     123.775     0.996094        71981       256.00
     124.351     0.996484        72008       284.44
     135.935     0.996875        72033       320.00
     140.415     0.997266        72067       365.71
     144.639     0.997656        72091       426.67
     148.991     0.998047        72117       512.00
     153.727     0.998242        72131       568.89
     157.567     0.998437        72147       640.00
     157.823     0.998633        72161       731.43
     161.919     0.998828        72174       853.33
     163.327     0.999023        72189      1024.00
     165.375     0.999121        72195      1137.78
     168.575     0.999219        72205      1280.00
     168.703     0.999316        72211      1462.86
     178.175     0.999414        72216      1706.67
     183.807     0.999512        72223      2048.00
     185.855     0.999561        72227      2275.56
     186.751     0.999609        72235      2560.00
     186.751     0.999658        72235      2925.71
     186.879     0.999707        72238      3413.33
     188.671     0.999756        72241      4096.00
     191.871     0.999780        72245      4551.11
     191.871     0.999805        72245      5120.00
     192.255     0.999829        72246      5851.43
     192.383     0.999854        72248      6826.67
     197.631     0.999878        72250      8192.00
     197.759     0.999890        72251      9102.22
     197.759     0.999902        72251     10240.00
     199.039     0.999915        72252     11702.86
     203.647     0.999927        72253     13653.33
     204.799     0.999939        72254     16384.00
     210.687     0.999945        72255     18204.44
     210.687     0.999951        72255     20480.00
     210.687     0.999957        72255     23405.71
     210.815     0.999963        72256     27306.67
     210.815     0.999969        72256     32768.00
     219.007     0.999973        72257     36408.89
     219.007     0.999976        72257     40960.00
     219.007     0.999979        72257     46811.43
     219.007     0.999982        72257     54613.33
     219.007     0.999985        72257     65536.00
     228.095     0.999986        72258     72817.78
     228.095     1.000000        72258          inf
#[Mean    =        8.234, StdDeviation   =       15.597]
#[Max     =      227.968, Total count    =        72258]
#[Buckets =           27, SubBuckets     =         2048]
----------------------------------------------------------
  86682 requests in 1.00m, 42.33MB read
  Socket errors: connect 0, read 0, write 0, timeout 14434
Requests/sec:   1442.43
Transfer/sec:    721.29KB
```

{{< /expand >}}

#### 测试结果

将结果中的延迟数据分为两组，我们先看第一组，分别保存为 `nginx.log` 和 `gateway.log` 文件中，并使用可视化工具：http://hdrhistogram.github.io/HdrHistogram/plotFiles.html，显示如图：

{{% load-img "/img/nginx_vs_gateway_latency.png" "" %}}

从图上可以看到在只处理 3000 并发的情况下，Gateway 的延迟分布非常稳定，并且延迟比 Nginx 要低不少，相反 Nginx 则表现略糟糕！

我们再来看看第二组延迟数据，这组主要是没有对延迟进行校正(未考虑延迟启动的测量)，同样方式生成可视化图表，如下图：

{{% load-img "/img/nginx_vs_gateway_latency_at_c1000-1.png" "" %}}

尽管 Wrk 统计延迟数据有一些区别，不过可以确定的是，在限制吞吐比较请求延迟的情况下，Gateway 依然完胜 Nginx。

不如，降低一下测试的并发，继续测试一下，结果如下：

{{< expand "展开查看详细压测结果" "..." >}}

```
root@loadgen:/opt/wrk2# wrk -t10 -c100 -d60s -R 3000 --u_latency  http://192.168.3.200:9090/test/_search
Running 1m test @ http://192.168.3.200:9090/test/_search
  10 threads and 100 connections
  Thread calibration: mean lat.: 2.856ms, rate sampling interval: 10ms
  Thread calibration: mean lat.: 2.899ms, rate sampling interval: 10ms
  Thread calibration: mean lat.: 2.914ms, rate sampling interval: 10ms
  Thread calibration: mean lat.: 2.952ms, rate sampling interval: 10ms
  Thread calibration: mean lat.: 2.945ms, rate sampling interval: 10ms
  Thread calibration: mean lat.: 2.938ms, rate sampling interval: 10ms
  Thread calibration: mean lat.: 2.890ms, rate sampling interval: 10ms
  Thread calibration: mean lat.: 2.918ms, rate sampling interval: 10ms
  Thread calibration: mean lat.: 2.939ms, rate sampling interval: 10ms
  Thread calibration: mean lat.: 2.909ms, rate sampling interval: 10ms
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     7.27s     5.73s   19.86s    51.19%
    Req/Sec   210.06    172.16     1.33k    57.97%
  Latency Distribution (HdrHistogram - Recorded Latency)
 50.000%    7.52s
 75.000%   12.03s
 90.000%   15.11s
 99.000%   17.91s
 99.900%   19.17s
 99.990%   19.81s
 99.999%   19.87s
100.000%   19.87s

  Detailed Percentile spectrum:
       Value   Percentile   TotalCount 1/(1-Percentile)

       1.327     0.000000            1         1.00
       2.373     0.100000         9980         1.11
       2.973     0.200000        19922         1.25
    2289.663     0.300000        29854         1.43
    5648.383     0.400000        39811         1.67
    7520.255     0.500000        49763         2.00
    8462.335     0.550000        54769         2.22
    9314.303     0.600000        59717         2.50
   10223.615     0.650000        64704         2.86
   11116.543     0.700000        69657         3.33
   12034.047     0.750000        74637         4.00
   12533.759     0.775000        77146         4.44
   12976.127     0.800000        79633         5.00
   13393.919     0.825000        82102         5.71
   13918.207     0.850000        84584         6.67
   14508.031     0.875000        87090         8.00
   14811.135     0.887500        88350         8.89
   15106.047     0.900000        89592        10.00
   15359.999     0.912500        90803        11.43
   15687.679     0.925000        92052        13.33
   16048.127     0.937500        93301        16.00
   16220.159     0.943750        93925        17.78
   16383.999     0.950000        94546        20.00
   16547.839     0.956250        95162        22.86
   16752.639     0.962500        95800        26.67
   16973.823     0.968750        96409        32.00
   17104.895     0.971875        96740        35.56
   17219.583     0.975000        97045        40.00
   17350.655     0.978125        97364        45.71
   17465.343     0.981250        97665        53.33
   17580.031     0.984375        97956        64.00
   17661.951     0.985938        98130        71.11
   17743.871     0.987500        98280        80.00
   17842.175     0.989062        98433        91.43
   17956.863     0.990625        98577       106.67
   18104.319     0.992188        98746       128.00
   18169.855     0.992969        98818       142.22
   18235.391     0.993750        98896       160.00
   18317.311     0.994531        98971       182.86
   18399.231     0.995313        99048       213.33
   18481.151     0.996094        99121       256.00
   18546.687     0.996484        99162       284.44
   18612.223     0.996875        99200       320.00
   18710.527     0.997266        99243       365.71
   18792.447     0.997656        99282       426.67
   18874.367     0.998047        99320       512.00
   18923.519     0.998242        99339       568.89
   18972.671     0.998437        99360       640.00
   19021.823     0.998633        99376       731.43
   19103.743     0.998828        99394       853.33
   19185.663     0.999023        99414      1024.00
   19218.431     0.999121        99424      1137.78
   19267.583     0.999219        99435      1280.00
   19300.351     0.999316        99442      1462.86
   19365.887     0.999414        99452      1706.67
   19447.807     0.999512        99461      2048.00
   19496.959     0.999561        99466      2275.56
   19546.111     0.999609        99471      2560.00
   19578.879     0.999658        99475      2925.71
   19628.031     0.999707        99481      3413.33
   19677.183     0.999756        99487      4096.00
   19693.567     0.999780        99489      4551.11
   19726.335     0.999805        99490      5120.00
   19742.719     0.999829        99492      5851.43
   19775.487     0.999854        99496      6826.67
   19791.871     0.999878        99498      8192.00
   19808.255     0.999890        99500      9102.22
   19808.255     0.999902        99500     10240.00
   19824.639     0.999915        99501     11702.86
   19841.023     0.999927        99506     13653.33
   19841.023     0.999939        99506     16384.00
   19841.023     0.999945        99506     18204.44
   19841.023     0.999951        99506     20480.00
   19841.023     0.999957        99506     23405.71
   19841.023     0.999963        99506     27306.67
   19841.023     0.999969        99506     32768.00
   19857.407     0.999973        99507     36408.89
   19857.407     0.999976        99507     40960.00
   19857.407     0.999979        99507     46811.43
   19873.791     0.999982        99509     54613.33
   19873.791     1.000000        99509          inf
#[Mean    =     7267.864, StdDeviation   =     5727.868]
#[Max     =    19857.408, Total count    =        99509]
#[Buckets =           27, SubBuckets     =         2048]
----------------------------------------------------------

  Latency Distribution (HdrHistogram - Uncorrected Latency (measured without taking delayed starts into account))
 50.000%   45.15ms
 75.000%   56.67ms
 90.000%   70.40ms
 99.000%  146.82ms
 99.900%  219.65ms
 99.990%  254.59ms
 99.999%  270.85ms
100.000%  306.69ms

  Detailed Percentile spectrum:
       Value   Percentile   TotalCount 1/(1-Percentile)

       1.209     0.000000            1         1.00
       1.670     0.100000         9981         1.11
       1.997     0.200000        19911         1.25
      27.519     0.300000        29853         1.43
      39.391     0.400000        39867         1.67
      45.151     0.500000        49755         2.00
      47.615     0.550000        54787         2.22
      49.919     0.600000        59747         2.50
      52.031     0.650000        64747         2.86
      54.143     0.700000        69660         3.33
      56.671     0.750000        74667         4.00
      58.207     0.775000        77146         4.44
      59.711     0.800000        79614         5.00
      61.503     0.825000        82110         5.71
      63.711     0.850000        84605         6.67
      66.367     0.875000        87079         8.00
      68.159     0.887500        88363         8.89
      70.463     0.900000        89599        10.00
      73.151     0.912500        90817        11.43
      77.503     0.925000        92049        13.33
      85.055     0.937500        93298        16.00
      92.287     0.943750        93912        17.78
     104.639     0.950000        94536        20.00
     115.967     0.956250        95157        22.86
     120.511     0.962500        95784        26.67
     124.543     0.968750        96401        32.00
     125.631     0.971875        96719        35.56
     128.319     0.975000        97031        40.00
     130.559     0.978125        97341        45.71
     132.863     0.981250        97650        53.33
     137.087     0.984375        97965        64.00
     138.879     0.985938        98120        71.11
     140.031     0.987500        98267        80.00
     144.383     0.989062        98422        91.43
     149.375     0.990625        98577       106.67
     156.031     0.992188        98735       128.00
     163.583     0.992969        98824       142.22
     165.503     0.993750        98888       160.00
     169.215     0.994531        98965       182.86
     174.079     0.995313        99043       213.33
     179.839     0.996094        99122       256.00
     181.375     0.996484        99160       284.44
     183.807     0.996875        99201       320.00
     187.263     0.997266        99237       365.71
     195.455     0.997656        99282       426.67
     201.215     0.998047        99315       512.00
     205.951     0.998242        99336       568.89
     208.127     0.998437        99354       640.00
     209.023     0.998633        99378       731.43
     214.527     0.998828        99393       853.33
     220.415     0.999023        99413      1024.00
     222.847     0.999121        99423      1137.78
     223.487     0.999219        99434      1280.00
     224.767     0.999316        99444      1462.86
     234.239     0.999414        99451      1706.67
     239.487     0.999512        99468      2048.00
     239.487     0.999561        99468      2275.56
     243.583     0.999609        99471      2560.00
     244.863     0.999658        99476      2925.71
     246.783     0.999707        99480      3413.33
     248.319     0.999756        99485      4096.00
     252.415     0.999780        99488      4551.11
     254.079     0.999805        99493      5120.00
     254.079     0.999829        99493      5851.43
     254.207     0.999854        99495      6826.67
     254.591     0.999878        99501      8192.00
     254.591     0.999890        99501      9102.22
     254.591     0.999902        99501     10240.00
     254.591     0.999915        99501     11702.86
     256.127     0.999927        99502     13653.33
     256.255     0.999939        99503     16384.00
     261.247     0.999945        99504     18204.44
     261.503     0.999951        99505     20480.00
     261.503     0.999957        99505     23405.71
     267.263     0.999963        99506     27306.67
     267.263     0.999969        99506     32768.00
     270.079     0.999973        99507     36408.89
     270.079     0.999976        99507     40960.00
     270.079     0.999979        99507     46811.43
     270.847     0.999982        99508     54613.33
     270.847     0.999985        99508     65536.00
     270.847     0.999986        99508     72817.78
     270.847     0.999988        99508     81920.00
     270.847     0.999989        99508     93622.86
     306.687     0.999991        99509    109226.67
     306.687     1.000000        99509          inf
#[Mean    =       41.878, StdDeviation   =       32.895]
#[Max     =      306.432, Total count    =        99509]
#[Buckets =           27, SubBuckets     =         2048]
----------------------------------------------------------
  129650 requests in 1.00m, 57.37MB read
Requests/sec:   2160.75
Transfer/sec:      0.96MB
```

{{< /expand >}}

需要注意的是，在压测过程中，被测机的 CPU 达到 100%，意味着服务器资源被充分利用，如下图：

{{% load-img "/img/nginx_latency_at_c100.png" "" %}}

#### 测试结果

相同参数测试 Gateway，如下：

{{< expand "展开查看详细压测结果" "..." >}}

```
root@loadgen:/opt/wrk2# wrk -t10 -c100 -d60s -R 3000 --u_latency  http://192.168.3.200:8000/test/_search
Running 1m test @ http://192.168.3.200:8000/test/_search
  10 threads and 100 connections
  Thread calibration: mean lat.: 8.889ms, rate sampling interval: 34ms
  Thread calibration: mean lat.: 8.887ms, rate sampling interval: 34ms
  Thread calibration: mean lat.: 5.895ms, rate sampling interval: 26ms
  Thread calibration: mean lat.: 5.599ms, rate sampling interval: 26ms
  Thread calibration: mean lat.: 6.167ms, rate sampling interval: 27ms
  Thread calibration: mean lat.: 4.795ms, rate sampling interval: 24ms
  Thread calibration: mean lat.: 6.096ms, rate sampling interval: 26ms
  Thread calibration: mean lat.: 6.002ms, rate sampling interval: 26ms
  Thread calibration: mean lat.: 8.079ms, rate sampling interval: 33ms
  Thread calibration: mean lat.: 5.316ms, rate sampling interval: 25ms
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     7.24ms    9.27ms 169.47ms   93.44%
    Req/Sec   162.83     83.56     1.25k    70.79%
  Latency Distribution (HdrHistogram - Recorded Latency)
 50.000%    4.23ms
 75.000%   10.96ms
 90.000%   14.62ms
 99.000%   35.52ms
 99.900%  122.69ms
 99.990%  152.06ms
 99.999%  167.93ms
100.000%  169.60ms

  Detailed Percentile spectrum:
       Value   Percentile   TotalCount 1/(1-Percentile)

       0.768     0.000000            1         1.00
       1.553     0.100000         7949         1.11
       1.846     0.200000        15903         1.25
       2.163     0.300000        23831         1.43
       2.719     0.400000        31772         1.67
       4.231     0.500000        39714         2.00
       5.675     0.550000        43681         2.22
       6.951     0.600000        47648         2.50
       8.199     0.650000        51623         2.86
       9.575     0.700000        55589         3.33
      10.959     0.750000        59571         4.00
      11.695     0.775000        61559         4.44
      12.239     0.800000        63551         5.00
      12.839     0.825000        65532         5.71
      13.327     0.850000        67500         6.67
      13.679     0.875000        69506         8.00
      14.055     0.887500        70484         8.89
      14.623     0.900000        71487        10.00
      15.271     0.912500        72465        11.43
      15.975     0.925000        73458        13.33
      16.751     0.937500        74453        16.00
      17.103     0.943750        74968        17.78
      17.263     0.950000        75474        20.00
      17.343     0.956250        76080        22.86
      17.503     0.962500        76477        26.67
      17.839     0.968750        76968        32.00
      18.063     0.971875        77178        35.56
      18.415     0.975000        77427        40.00
      19.007     0.978125        77676        45.71
      20.383     0.981250        77923        53.33
      23.487     0.984375        78171        64.00
      25.503     0.985938        78294        71.11
      29.183     0.987500        78418        80.00
      32.831     0.989062        78542        91.43
      37.247     0.990625        78666       106.67
      43.615     0.992188        78790       128.00
      48.895     0.992969        78852       142.22
      56.095     0.993750        78914       160.00
      62.911     0.994531        78976       182.86
      70.719     0.995313        79038       213.33
      80.575     0.996094        79100       256.00
      85.695     0.996484        79132       284.44
      91.071     0.996875        79162       320.00
      95.487     0.997266        79193       365.71
     100.607     0.997656        79224       426.67
     105.599     0.998047        79255       512.00
     109.119     0.998242        79271       568.89
     112.191     0.998437        79286       640.00
     116.159     0.998633        79302       731.43
     119.423     0.998828        79318       853.33
     122.943     0.999023        79334      1024.00
     126.207     0.999121        79342      1137.78
     127.871     0.999219        79348      1280.00
     130.879     0.999316        79356      1462.86
     133.119     0.999414        79364      1706.67
     136.703     0.999512        79372      2048.00
     137.471     0.999561        79376      2275.56
     138.239     0.999609        79379      2560.00
     142.079     0.999658        79383      2925.71
     143.743     0.999707        79387      3413.33
     144.767     0.999756        79391      4096.00
     145.151     0.999780        79393      4551.11
     145.919     0.999805        79395      5120.00
     147.455     0.999829        79397      5851.43
     149.631     0.999854        79399      6826.67
     150.783     0.999878        79401      8192.00
     152.063     0.999890        79402      9102.22
     155.135     0.999902        79403     10240.00
     155.903     0.999915        79404     11702.86
     158.463     0.999927        79405     13653.33
     158.975     0.999939        79406     16384.00
     158.975     0.999945        79406     18204.44
     160.895     0.999951        79407     20480.00
     160.895     0.999957        79407     23405.71
     161.279     0.999963        79408     27306.67
     161.279     0.999969        79408     32768.00
     161.279     0.999973        79408     36408.89
     167.935     0.999976        79409     40960.00
     167.935     0.999979        79409     46811.43
     167.935     0.999982        79409     54613.33
     167.935     0.999985        79409     65536.00
     167.935     0.999986        79409     72817.78
     169.599     0.999988        79410     81920.00
     169.599     1.000000        79410          inf
#[Mean    =        7.236, StdDeviation   =        9.266]
#[Max     =      169.472, Total count    =        79410]
#[Buckets =           27, SubBuckets     =         2048]
----------------------------------------------------------

  Latency Distribution (HdrHistogram - Uncorrected Latency (measured without taking delayed starts into account))
 50.000%    1.52ms
 75.000%    2.14ms
 90.000%    3.64ms
 99.000%   20.93ms
 99.900%   96.89ms
 99.990%  144.13ms
 99.999%  164.61ms
100.000%  168.06ms

  Detailed Percentile spectrum:
       Value   Percentile   TotalCount 1/(1-Percentile)

       0.712     0.000000            1         1.00
       1.091     0.100000         7952         1.11
       1.242     0.200000        15891         1.25
       1.332     0.300000        23831         1.43
       1.415     0.400000        31765         1.67
       1.521     0.500000        39735         2.00
       1.584     0.550000        43707         2.22
       1.661     0.600000        47651         2.50
       1.762     0.650000        51642         2.86
       1.910     0.700000        55596         3.33
       2.137     0.750000        59567         4.00
       2.291     0.775000        61543         4.44
       2.471     0.800000        63545         5.00
       2.669     0.825000        65518         5.71
       2.963     0.850000        67512         6.67
       3.317     0.875000        69491         8.00
       3.491     0.887500        70483         8.89
       3.645     0.900000        71477        10.00
       3.805     0.912500        72472        11.43
       4.007     0.925000        73458        13.33
       5.827     0.937500        74447        16.00
      12.815     0.943750        75309        17.78
      12.823     0.950000        76143        20.00
      12.823     0.956250        76143        22.86
      12.831     0.962500        76605        26.67
      13.415     0.968750        76930        32.00
      15.063     0.971875        77177        35.56
      16.815     0.975000        77475        40.00
      16.831     0.978125        78162        45.71
      16.831     0.981250        78162        53.33
      16.847     0.984375        78298        64.00
      16.847     0.985938        78298        71.11
      17.839     0.987500        78449        80.00
      19.839     0.989062        78559        91.43
      22.319     0.990625        78666       106.67
      23.983     0.992188        78797       128.00
      24.031     0.992969        78852       142.22
      24.847     0.993750        78920       160.00
      26.367     0.994531        78977       182.86
      28.495     0.995313        79038       213.33
      32.271     0.996094        79100       256.00
      35.679     0.996484        79131       284.44
      39.295     0.996875        79162       320.00
      43.359     0.997266        79193       365.71
      47.999     0.997656        79230       426.67
      52.031     0.998047        79255       512.00
      55.967     0.998242        79271       568.89
      65.279     0.998437        79286       640.00
      76.031     0.998633        79305       731.43
      83.903     0.998828        79317       853.33
      97.087     0.999023        79333      1024.00
     103.999     0.999121        79341      1137.78
     108.671     0.999219        79348      1280.00
     114.687     0.999316        79356      1462.86
     119.743     0.999414        79365      1706.67
     123.967     0.999512        79372      2048.00
     124.351     0.999561        79376      2275.56
     127.871     0.999609        79379      2560.00
     130.431     0.999658        79383      2925.71
     134.143     0.999707        79387      3413.33
     136.063     0.999756        79392      4096.00
     136.447     0.999780        79393      4551.11
     137.215     0.999805        79395      5120.00
     138.111     0.999829        79397      5851.43
     143.999     0.999854        79399      6826.67
     144.127     0.999878        79403      8192.00
     144.127     0.999890        79403      9102.22
     144.127     0.999902        79403     10240.00
     144.255     0.999915        79404     11702.86
     149.119     0.999927        79405     13653.33
     149.887     0.999939        79407     16384.00
     149.887     0.999945        79407     18204.44
     149.887     0.999951        79407     20480.00
     149.887     0.999957        79407     23405.71
     154.367     0.999963        79408     27306.67
     154.367     0.999969        79408     32768.00
     154.367     0.999973        79408     36408.89
     164.607     0.999976        79409     40960.00
     164.607     0.999979        79409     46811.43
     164.607     0.999982        79409     54613.33
     164.607     0.999985        79409     65536.00
     164.607     0.999986        79409     72817.78
     168.063     0.999988        79410     81920.00
     168.063     1.000000        79410          inf
#[Mean    =        2.790, StdDeviation   =        5.839]
#[Max     =      167.936, Total count    =        79410]
#[Buckets =           27, SubBuckets     =         2048]
----------------------------------------------------------
  95377 requests in 1.00m, 46.57MB read
  Socket errors: connect 0, read 0, write 0, timeout 1357
Requests/sec:   1589.41
Transfer/sec:    794.71KB

```

{{< /expand >}}

{{% load-img "/img/nginx_vs_gateway_latency_at_c100.png" "" %}}
{{% load-img "/img/nginx_vs_gateway_latency_at_c100-1.png" "" %}}

随着并发的降低，总体的延迟有所降低，不过 Gateway 依然碾压 Nginx。

## 写入压力测试

### 写入测试方法

由于 wrk 不支持随机构造 bulk 请求，我们这里使用另外一个工具 Loadgen 来进行写入的压力测试，详细的文档说明：[Loadgen](./benchmark.md)。

### 配置 Loadgen

我们配置一个每个批次 1000 文档的 bulk 请求，并使用 Loadgen 的随机变量来填充每个文档，让写入变得随机。

```
root@loadgen:/opt/loadgen# cat cfg/loadgen-200-nginx.yml
statsd:
  enabled: false
variables:
  - name: ip
    type: file
    path: test/ip.txt
  - name: id
    type: sequence
  - name: uuid
    type: uuid
  - name: now_local
    type: now_local
  - name: now_utc
    type: now_utc
  - name: now_unix
    type: now_unix
requests:
  - request:
      has_variable: true
      method: POST
      url: http://192.168.3.200:8000/_bulk
      body_repeat_times: 1000
      body: "{ \"create\" : { \"_index\" : \"loadgen_test\",\"_type\":\"_doc\", \"_id\" : \"$[[uuid]]\" } }\n{ \"id\" : \"$[[uuid]]\",\"field1\" : \"$[[user]]\",\"ip\" : \"$[[ip]]\",\"now_local\" : \"$[[now_local]]\",\"now_unix\" : \"$[[now_unix]]\" }\n"
```

我们分别修改 URL 参数为 `http://192.168.3.200:9090/_bulk` 和 `http://192.168.3.200:8000/_bulk` 来代表压测 Nginx 和 Gateway。

### 执行测试

删除索引 `loadgen_test`，并进行初始化，如下:

```
DELETE loadgen_test
PUT loadgen_test
{
  "settings": {
    "number_of_shards": 3,
    "number_of_replicas": 0,
    "refresh_interval": "10s",
    "index.translog.durability":"async"
  }
}
```

使用 100 个并发，持续时间 60s，执行以下命令来进行压测：

```
root@loadgen:/opt/loadgen# ./loadgen-linux-amd64 -config cfg/loadgen-200-nginx.yml -c 100 -d 60
```

#### 执行压测

首先，我们看一下 Nginx 的压测结果，如下：

```
root@loadgen:/opt/loadgen# ./loadgen-linux-amd64 -config cfg/loadgen-200-nginx.yml -c 100 -d 60
...

6887 requests in 20.032150884s, 1.63GB sent, 1.34GB received

[Loadgen Client Metrics]
Requests/sec:		114.78
Request Traffic/sec:	27.79MB
Total Transfer/sec:	50.73MB
Avg Req Time:		8.712066ms
Fastest Request:	28.131575ms
Slowest Request:	4.041278181s
Number of Errors:	0
Number of Invalid:	0
Status 200:		6887

[Estimated Server Metrics]
Requests/sec:		343.80
Transfer/sec:		151.95MB
Avg Req Time:		290.869041ms
```

清理索引，继续对 Gateway 进行压测，结果如下：

```
root@loadgen:/opt/loadgen# ./loadgen-linux-amd64 -config cfg/loadgen-200.yml -c 100 -d 60
...
7443 requests in 20.002412954s, 1.76GB sent, 1.42GB received

[Loadgen Client Metrics]
Requests/sec:		124.05
Request Traffic/sec:	30.04MB
Total Transfer/sec:	54.26MB
Avg Req Time:		8.061265ms
Fastest Request:	663.191µs
Slowest Request:	4.485718128s
Number of Errors:	172
Number of Invalid:	0
Status 200:		7271
Status 0:		172

[Estimated Server Metrics]
Requests/sec:		372.11
Transfer/sec:		162.75MB
Avg Req Time:		268.741273ms
```

Gateway 的吞吐为 `124.05` 略快于 Nginx 的 `114.78`。

监控显示 Loadgen 和 Elasticsearch 的 CPU 都已经跑满，而被压测节点 CPU 还比较空闲，说明当前代理转发能力还没充分发挥。

{{% load-img "/img/loadgen_c200_load.png" "" %}}
{{% load-img "/img/es_c200_load.png" "" %}}

鉴于 Elasticsearch 资源吃的太满，我们去掉有些优化参数，适当降低一下后端的处理能力，我们修改索引重建操作为：

```
DELETE loadgen_test
PUT loadgen_test
{
  "settings": {
    "number_of_shards": 3,
    "number_of_replicas": 1,
    "refresh_interval": "1s"
  }
}
```

我们降低批次文档为 10，调整并发为 400 再次进行测试。
这次我们先测 Gateway，清理索引，执行命令：

```
root@loadgen:/opt/loadgen# ./loadgen-linux-amd64 -config cfg/loadgen-200.yml -c 400 -d 60
   __   ___  _      ___  ___   __    __
  / /  /___\/_\    /   \/ _ \ /__\/\ \ \
 / /  //  ///_\\  / /\ / /_\//_\ /  \/ /
/ /__/ \_//  _  \/ /_// /_\\//__/ /\  /
\____|___/\_/ \_/___,'\____/\__/\_\ \/

[LOADGEN] A http load generator and testing suit.
[LOADGEN] 1.0.0_SNAPSHOT, 2022-04-06 04:17:50, 2023-12-31 10:10:10, 752b7d6233712a81320500fb7269fac4a89609c6
[04-24 07:30:32] [INF] [app.go:174] initializing loadgen.
[04-24 07:30:32] [INF] [app.go:175] using config: /opt/loadgen/cfg/loadgen-200.yml.
[04-24 07:30:32] [INF] [module.go:116] all modules are started
[04-24 07:30:33] [INF] [instance.go:72] workspace: /opt/loadgen/data/loadgen/nodes/c95unj9k09mm6eaee7m0
[04-24 07:30:33] [INF] [app.go:283] loadgen is up and running now.
[04-24 07:30:33] [INF] [loader.go:315] warmup started
[04-24 07:30:33] [INF] [loader.go:328] [POST] http://192.168.3.200:8000/_bulk
[04-24 07:30:33] [INF] [loader.go:329] status: 200,<nil>,{"took":122,"errors":false,"items":[{"create":{"_index":"loadgen_test","_type":"_doc","_id":"c9ifp69k09mh92f52nu0","_version":1,"result":"created","_shards":{"total":2,"successful":1,"failed":0},"_seq_no":0,"_primary_term":1,"status":201}},{"create":{"_ind
[04-24 07:30:33] [INF] [loader.go:337] warmup finished

319600 requests in 50.90639728s, 773.84MB sent, 647.78MB received

[Loadgen Client Metrics]
Requests/sec:		5326.67
Request Traffic/sec:	12.90MB
Total Transfer/sec:	23.69MB
Avg Req Time:		187.734µs
Fastest Request:	4.043585ms
Slowest Request:	3.116303008s
Number of Errors:	11
Number of Invalid:	0
Status 200:		319589
Status 0:		11

[Estimated Server Metrics]
Requests/sec:		6278.19
Transfer/sec:		27.93MB
Avg Req Time:		63.712637ms
```

测试 Nginx，清理索引，执行命令：

```
root@loadgen:/opt/loadgen# ./loadgen-linux-amd64 -config cfg/loadgen-200-nginx.yml -c 400 -d 60
   __   ___  _      ___  ___   __    __
  / /  /___\/_\    /   \/ _ \ /__\/\ \ \
 / /  //  ///_\\  / /\ / /_\//_\ /  \/ /
/ /__/ \_//  _  \/ /_// /_\\//__/ /\  /
\____|___/\_/ \_/___,'\____/\__/\_\ \/

[LOADGEN] A http load generator and testing suit.
[LOADGEN] 1.0.0_SNAPSHOT, 2022-04-06 04:17:50, 2023-12-31 10:10:10, 752b7d6233712a81320500fb7269fac4a89609c6
[04-24 07:33:30] [INF] [app.go:174] initializing loadgen.
[04-24 07:33:30] [INF] [app.go:175] using config: /opt/loadgen/cfg/loadgen-200-nginx.yml.
[04-24 07:33:30] [INF] [module.go:116] all modules are started
[04-24 07:33:32] [INF] [instance.go:72] workspace: /opt/loadgen/data/loadgen/nodes/c95unj9k09mm6eaee7m0
[04-24 07:33:32] [INF] [app.go:283] loadgen is up and running now.
[04-24 07:33:32] [INF] [loader.go:315] warmup started
[04-24 07:33:32] [INF] [loader.go:328] [POST] http://192.168.3.200:9090/_bulk
[04-24 07:33:32] [INF] [loader.go:329] status: 200,<nil>,{"took":118,"errors":false,"items":[{"create":{"_index":"loadgen_test","_type":"_doc","_id":"c9ifqj1k09mh95ie91g0","_version":1,"result":"created","_shards":{"total":2,"successful":1,"failed":0},"_seq_no":0,"_primary_term":1,"status":201}},{"create":{"_ind
[04-24 07:33:32] [INF] [loader.go:337] warmup finished

166454 requests in 59.708162329s, 403.03MB sent, 337.07MB received

[Loadgen Client Metrics]
Requests/sec:		2774.23
Request Traffic/sec:	6.72MB
Total Transfer/sec:	12.34MB
Avg Req Time:		360.459µs
Fastest Request:	7.899973ms
Slowest Request:	7.271097939s
Number of Errors:	0
Number of Invalid:	0
Status 200:		166454

[Estimated Server Metrics]
Requests/sec:		2787.79
Transfer/sec:		12.40MB
Avg Req Time:		143.482673ms
```

上面的压测结果输出显示，Gateway 的写吞吐为 Nginx 的 2 倍多，从 Elasticsearch 后台的监控也可以看到，Gateway 的转发效率要明显高于 Nginx：

{{% load-img "/img/nginx_vs_gateway_c400_d10.png" "" %}}

从机器资源来看，被测端的 CPU 利用率上去了，压测端还有一些余量，说明压力都给上去了。

{{% load-img "/img/nginx_c400_d10_load.png" "" %}}

从 Elasticsearch 的资源来看，也还有一些余量，如下图:
{{% load-img "/img/nginx_c400_d10_es_load.png" "" %}}

继续调整批次文档为 1，加大写入并发为 500，测试如下:

```
root@loadgen:/opt/loadgen# ./loadgen-linux-amd64 -config cfg/loadgen-200.yml -c 500 -d 60
   __   ___  _      ___  ___   __    __
  / /  /___\/_\    /   \/ _ \ /__\/\ \ \
 / /  //  ///_\\  / /\ / /_\//_\ /  \/ /
/ /__/ \_//  _  \/ /_// /_\\//__/ /\  /
\____|___/\_/ \_/___,'\____/\__/\_\ \/

[LOADGEN] A http load generator and testing suit.
[LOADGEN] 1.0.0_SNAPSHOT, 2022-04-06 04:17:50, 2023-12-31 10:10:10, 752b7d6233712a81320500fb7269fac4a89609c6
[04-24 07:47:48] [INF] [app.go:174] initializing loadgen.
[04-24 07:47:48] [INF] [app.go:175] using config: /opt/loadgen/cfg/loadgen-200.yml.
[04-24 07:47:48] [INF] [module.go:116] all modules are started
[04-24 07:47:49] [INF] [instance.go:72] workspace: /opt/loadgen/data/loadgen/nodes/c95unj9k09mm6eaee7m0
[04-24 07:47:49] [INF] [app.go:283] loadgen is up and running now.
[04-24 07:47:49] [INF] [loader.go:315] warmup started
[04-24 07:47:50] [INF] [loader.go:328] [POST] http://192.168.3.200:8000/_bulk
[04-24 07:47:50] [INF] [loader.go:329] status: 200,<nil>,{"took":100,"errors":false,"items":[{"create":{"_index":"loadgen_test","_type":"_doc","_id":"c9ig199k09mh9k3i5m00","_version":1,"result":"created","_shards":{"total":2,"successful":1,"failed":0},"_seq_no":0,"_primary_term":1,"status":201}}]}
[04-24 07:47:50] [INF] [loader.go:337] warmup finished

909281 requests in 59.536624095s, 220.16MB sent, 211.47MB received

[Loadgen Client Metrics]
Requests/sec:		15154.68
Request Traffic/sec:	3.67MB
Total Transfer/sec:	7.19MB
Avg Req Time:		65.986µs
Fastest Request:	2.052998ms
Slowest Request:	1.797984169s
Number of Errors:	16
Number of Invalid:	0
Status 200:		909265
Status 0:		16

[Estimated Server Metrics]
Requests/sec:		15272.63
Transfer/sec:		7.25MB
Avg Req Time:		32.738297ms

root@loadgen:/opt/loadgen# vi cfg/loadgen-200-nginx.yml
root@loadgen:/opt/loadgen# ./loadgen-linux-amd64 -config cfg/loadgen-200-nginx.yml -c 500 -d 60
   __   ___  _      ___  ___   __    __
  / /  /___\/_\    /   \/ _ \ /__\/\ \ \
 / /  //  ///_\\  / /\ / /_\//_\ /  \/ /
/ /__/ \_//  _  \/ /_// /_\\//__/ /\  /
\____|___/\_/ \_/___,'\____/\__/\_\ \/

[LOADGEN] A http load generator and testing suit.
[LOADGEN] 1.0.0_SNAPSHOT, 2022-04-06 04:17:50, 2023-12-31 10:10:10, 752b7d6233712a81320500fb7269fac4a89609c6
[04-24 07:49:34] [INF] [app.go:174] initializing loadgen.
[04-24 07:49:34] [INF] [app.go:175] using config: /opt/loadgen/cfg/loadgen-200-nginx.yml.
[04-24 07:49:34] [INF] [module.go:116] all modules are started
[04-24 07:49:36] [INF] [instance.go:72] workspace: /opt/loadgen/data/loadgen/nodes/c95unj9k09mm6eaee7m0
[04-24 07:49:36] [INF] [app.go:283] loadgen is up and running now.
[04-24 07:49:36] [INF] [loader.go:315] warmup started
[04-24 07:49:36] [INF] [loader.go:328] [POST] http://192.168.3.200:9090/_bulk
[04-24 07:49:36] [INF] [loader.go:329] status: 200,<nil>,{"took":93,"errors":false,"items":[{"create":{"_index":"loadgen_test","_type":"_doc","_id":"c9ig241k09mh9q44d5hg","_version":1,"result":"created","_shards":{"total":2,"successful":1,"failed":0},"_seq_no":0,"_primary_term":1,"status":201}}]}
[04-24 07:49:36] [INF] [loader.go:337] warmup finished

246921 requests in 1m0.081264783s, 59.79MB sent, 57.22MB received

[Loadgen Client Metrics]
Requests/sec:		4115.35
Request Traffic/sec:	1020.34KB
Total Transfer/sec:	1.95MB
Avg Req Time:		242.992µs
Fastest Request:	5.184094ms
Slowest Request:	31.658816982s
Number of Errors:	0
Number of Invalid:	0
Status 200:		246921

[Estimated Server Metrics]
Requests/sec:		4109.78
Transfer/sec:		1.95MB
Avg Req Time:		121.660905ms
```

从上面的结果来看，Gateway 和 Nginx 的差距进一步拉大，Gateway 的吞吐超过 Nginx 3 倍之多。

{{% load-img "/img/nginx_vs_gateway_c500_d1.png" "" %}}

从监控来看，被测端和 Elasticsearch 的 CPU 都达到 100%，已经达到测试极限最大值。
{{% load-img "/img/gateway_es_load_c500_d1.png" "" %}}

总体来看，在写入场景，Gateway 总体性能要远高于 Nginx。

## 深度调优 Nginx 再测

有网友反馈 Nginx 还有优化空间，系统 CPU 占比过高，经过检查发现 Nginx 的默认长链接处理有点问题，TCP 的 TIMEWAIT 比较高，

### 优化 Nginx 配置

看来 Nginx 除了前面的几个参数，还需要额外的一些调优，经过反复几轮的测试，Nginx 的 TIMEWAIT 清零了，具体优化配置如下：

{{< expand "展开查看详细 Nginx 配置" "..." >}}

```
root@gateway:/tmp# cat /etc/nginx/nginx.conf
user www-data;
worker_processes auto;
pid /run/nginx.pid;
include /etc/nginx/modules-enabled/*.conf;
worker_rlimit_nofile 10240;

events {
	worker_connections 10000;
	multi_accept on;
}

http {

	sendfile on;
	tcp_nopush on;
	tcp_nodelay on;

    keepalive_timeout  300s;
    keepalive_requests 1000000;
    gzip  off;
    access_log off;

	types_hash_max_size 2048;
	include /etc/nginx/mime.types;
	default_type application/octet-stream;

	ssl_protocols TLSv1 TLSv1.1 TLSv1.2; # Dropping SSLv3, ref: POODLE
	ssl_prefer_server_ciphers on;

	include /etc/nginx/conf.d/*.conf;
	include /etc/nginx/sites-enabled/*;
}

root@gateway:/tmp# cat /etc/nginx/conf.d/es.conf
upstream servers {
    server 192.168.3.188:9236;
    server 192.168.3.188:9226;
    server 192.168.3.188:9216;
    server 192.168.3.188:9206;
    keepalive 1000;
  }

server {
    listen 0.0.0.0:9090;
    access_log off;

location / {
    proxy_pass http://servers;

          proxy_set_header Connection "keep-alive";
          proxy_http_version 1.1;
          proxy_ignore_client_abort on;
         proxy_connect_timeout 600;
         proxy_read_timeout 600;
         proxy_send_timeout 600;
}
}
```

{{< /expand >}}

重启生效

### 重新测试查询

### 重新测试延迟

### 重新测试写入

# 总结

通过对 Nginx 和 INFINI Gateway 就 Elasticsearch 的写入和查询这两个典型场景的性能测试，相信大家对 INFINI Gateway 的性能情况有了一点的了解，
在上面的个别测试场景中，Gateway 的峰值吞吐甚至超越 Nginx 三倍之多，无论是从吞吐还是延迟方面，亦或是在高并发的处理能力上面，INFINI Gateway 都有非常不错的表现，
不过性能不代表一切，Nginx 也是非常优秀，功能也更丰富，极限网关也正在不断迭代，欢迎大家继续保持关注。
