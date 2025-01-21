---
title: "indexing_merge"
asciinema: false
---

# indexing_merge

## 描述

indexing_merge 处理器用来消费队列里面的纯 JSON 文档，并合并成 Bulk 请求保存到指定的队列里面，需要配合 `bulk_indexing` 处理器进行消费，用批量写入代替单次请求来提高写入吞吐。

## 配置示例

一个简单的示例如下：

```
pipeline:
  - name: indexing_merge
    auto_start: true
    keep_running: true
    processor:
      - indexing_merge:
          input_queue: "request_logging"
          elasticsearch: "logging-server"
          index_name: "infini_gateway_requests"
          output_queue:
            name: "gateway_requests"
            label:
              tag: "request_logging"
          worker_size: 1
          bulk_size_in_mb: 10
  - name: logging_requests
    auto_start: true
    keep_running: true
    processor:
      - bulk_indexing:
          bulk:
            compress: true
            batch_size_in_mb: 10
            batch_size_in_docs: 5000
          consumer:
            fetch_max_messages: 100
          queues:
            type: indexing_merge
          when:
            cluster_available: [ "logging-server" ]
```

## 参数说明

| 名称                    | 类型   | 说明                                                                                 |
| ----------------------- | ------ | ------------------------------------------------------------------------------------ |
| input_queue             | string | 订阅的队列名称                                                                       |
| worker_size             | int    | 并行执行消费任务的线程数，默认 `1`                                                   |
| idle_timeout_in_seconds | int    | 消费队列的超时时间，默认 `5`，单位秒                                                 |
| bulk_size_in_kb         | int    | 批次请求的单位大小，单位 `KB`                                                        |
| bulk_size_in_mb         | int    | 批次请求的单位大小，单位 `MB`，默认 `10`                                             |
| elasticsearch           | string | 保存到目标集群的名称                                                                 |
| index_name              | string | 保存到目标集群的索引名称                                                             |
| type_name               | string | 保存到目标集群的索引类型名称，默认根据集群版本来设置，v7 以前为 `doc`，之后为 `_doc` |
| output_queue.name       | string | 保存到目标队列的名称                                                                 |
| output_queue.label      | map    | 保存到目标队列的标签，内置 `type:indexing_merge`                                     |
| failure_queue           | string | 保存可重试的失败请求的队列名称                                                       |
| invalid_queue           | string | 保存不合法的失败请求的队列名称                                                       |
