---
title: "索引文档级别差异对比"
weight: 30
asciinema: true
---

# 索引差异对比

通过极限网关可以进行索引的文档差异对比，可以对同集群或者跨集群的两个不同的索引进行 diff 比较，对于使用应用双写、CCR 或者其他数据复制方案的场景，可以进行定期 diff 比较来确保数据是否真的一致。

## 功能演示

{{< asciinema key="/index_diff" speed="2"  autoplay="1"  start-at="0" rows="30" preload="1" >}}

## 如何配置

### 设置目标集群

修改配置文件 `gateway.yml`，设置两个集群资源 `source` 和 `target`，增加如下配置：

```
elasticsearch:
  - name: source
    enabled: true
    endpoint: http://localhost:9200
    basic_auth:
      username: test
      password: testtest
  - name: target
    enabled: true
    endpoint: http://localhost:9201
    basic_auth: #used to discovery full cluster nodes, or check elasticsearch's health and versions
      username: test
      password: testtest
```

### 配置对比任务

增加一个服务管道配置，用来处理两个集群的索引文档拉取和对比，如下：

```
pipeline:
  - name: index_diff_service
    auto_start: true
    keep_running: true
    processor:
    - dag:
        parallel:
          - dump_hash: #dump es1's doc
              indices: "medcl-test"
              scroll_time: "10m"
              elasticsearch: "source"
              output_queue: "source_docs"
              batch_size: 10000
              slice_size: 5
          - dump_hash: #dump es2's doc
              indices: "medcl-test"
              scroll_time: "10m"
              batch_size: 10000
              slice_size: 5
              elasticsearch: "target"
              output_queue: "target_docs"
        end:
          - index_diff:
              diff_queue: "diff_result"
              buffer_size: 1
              text_report: true #如果要存 es，这个开关关闭，开启 pipeline 的 diff_result_ingest 任务
              source_queue: 'source_docs'
              target_queue: 'target_docs'
```

上面的配置中，并行使用了 `dump_hash` 来拉取集群 `source` 的 `medcl-a` 索引和取集群 `target` 的 `medcl-b` 索引，并以文本结果的方式输出到终端。

### 输出结果到 Elasticsearch

如果 diff 结果比较多，可以选择保存到 `Elasticsearch` 集群，将上面的 `index_diff` 处理单元的参数 `text_report` 设置为 `false`，并增加如下配置：

```
pipeline:
  - name: diff_result_ingest
    auto_start: true
    keep_running: true
    processor:
      - json_indexing:
          index_name: "diff_result"
          elasticsearch: "source"
          input_queue: "diff_result"
          idle_timeout_in_seconds: 1
          worker_size: 1
          bulk_size_in_mb: 10 #in MB
```

最后导入[仪表板](https://github.com/medcl/infini-gateway/releases/download/1.2.0/index-diff-report-v7.12.ndjson.zip) 到 Kibana 即可看到如下效果：

{{% load-img "/img/index-diff-dashboard.jpg" "" %}}
