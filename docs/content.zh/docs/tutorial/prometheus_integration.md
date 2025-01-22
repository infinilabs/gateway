---
title: "与 Prometheus 集成"
weight: 100
---

# 与 Prometheus 集成

极限网关支持将运行指标输出为 Prometheus 格式, 方便与 Prometheus 进行集成, 具体操作如下:

## 统计信息接口

访问网关的 2900 接口,如下:

```
http://localhost:2900/stats?format=prometheus
➜  ~ curl http://localhost:2900/stats\?format\=prometheus
buffer_fasthttp_resbody_buffer_acquired{type="gateway", ip="192.168.3.23", name="Orchid", id="cbvjphrq50kcnsu2a8v0"} 1
buffer_stats_acquired{type="gateway", ip="192.168.3.23", name="Orchid", id="cbvjphrq50kcnsu2a8v0"} 7
buffer_stats_max_count{type="gateway", ip="192.168.3.23", name="Orchid", id="cbvjphrq50kcnsu2a8v0"} 0
system_cpu{type="gateway", ip="192.168.3.23", name="Orchid", id="cbvjphrq50kcnsu2a8v0"} 0
buffer_bulk_request_docs_acquired{type="gateway", ip="192.168.3.23", name="Orchid", id="cbvjphrq50kcnsu2a8v0"} 1
buffer_fasthttp_resbody_buffer_inuse{type="gateway", ip="192.168.3.23", name="Orchid", id="cbvjphrq50kcnsu2a8v0"} 0
stats_gateway_request_bytes{type="gateway", ip="192.168.3.23", name="Orchid", id="cbvjphrq50kcnsu2a8v0"} 0
system_mem{type="gateway", ip="192.168.3.23", name="Orchid", id="cbvjphrq50kcnsu2a8v0"} 31473664
...
```

通过增加额外的参数 `format=prometheus` 即可返回 Prometheus 所需数据格式.

## 配置 Prometheus 进行采集

修改配置文件: prometheus.yml

```
# my global config
global:
  scrape_interval: 15s # Set the scrape interval to every 15 seconds. Default is every 1 minute.
  evaluation_interval: 15s # Evaluate rules every 15 seconds. The default is every 1 minute.
  # scrape_timeout is set to the global default (10s).

# Alertmanager configuration
alerting:
  alertmanagers:
    - static_configs:
        - targets:
          # - alertmanager:9093

# Load rules once and periodically evaluate them according to the global 'evaluation_interval'.
rule_files:
  # - "first_rules.yml"
  # - "second_rules.yml"

# A scrape configuration containing exactly one endpoint to scrape:
# Here it's Prometheus itself.
scrape_configs:
  - job_name: "prometheus"
    scrape_interval: 5s
    # metrics_path defaults to '/metrics'
    metrics_path: /stats
    params:
      format: ['prometheus']
    # scheme defaults to 'http'.
    static_configs:
      - targets: ["localhost:2900"]
        labels:
          group: 'infini'
```

## 启动 Prometheus

启动之后,可以看到指标正常收集.

{{% load-img "/img/prometheus_target.png" "" %}}

然后就可以持续检测网关的运行状态了.

{{% load-img "/img/prometheus_graph.png" "" %}}
