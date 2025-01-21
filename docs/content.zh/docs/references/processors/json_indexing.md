---
title: "json_indexing"
asciinema: false
---

# json_indexing

## 描述

json_indexing 处理器用来消费队列里面的纯 JSON 文档，并保存到指定的 Elasticsearch 服务器里面。

## 配置示例

一个简单的示例如下：

```
pipeline:
- name: request_logging_index
  auto_start: true
  keep_running: true
  processor:
    - json_indexing:
        index_name: "gateway_requests"
        elasticsearch: "dev"
        input_queue: "request_logging"
        idle_timeout_in_seconds: 1
        worker_size: 1
        bulk_size_in_mb: 10
```

## 参数说明

| 名称                    | 类型   | 说明                                                                                 |
| ----------------------- | ------ | ------------------------------------------------------------------------------------ |
| input_queue             | string | 订阅的队列名称                                                                       |
| worker_size             | int    | 并行执行消费任务的线程数，默认 `1`                                                   |
| idle_timeout_in_seconds | int    | 消费队列的超时时间，默认 `5`，单位秒                                                 |
| bulk_size_in_kb         | int    | 批次请求的单位大小，单位 `KB`                                                        |
| bulk_size_in_mb         | int    | 批次请求的单位大小，单位 `MB`                                                        |
| elasticsearch           | string | 保存到目标集群的名称                                                                 |
| index_name              | string | 保存到目标集群的索引名称                                                             |
| type_name               | string | 保存到目标集群的索引类型名称，默认根据集群版本来设置，v7 以前为 `doc`，之后为 `_doc` |
