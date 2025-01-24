---
title: "离线处理器"
weight: 90
bookCollapseSection: true
---

# 服务管道

## 什么是服务管道

服务管道（Pipeline）是用于离线处理任务的功能组合，和在线请求的过滤器一样使用管道设计模式。
处理器（Processor）是服务管道的基础单位，每个处理组件一般专注做一件事情，根据需要灵活组装，灵活插拔。

## 管道定义

一个典型的管道服务定义如下：

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
        bulk_size_in_mb: 10 #in MB
```

上面的配置里面，定义了一个名为 `request_logging_index` 的处理管道，`processor` 参数定义了该管道的若干处理单元，依次执行。

## 参数说明

管道定义的相关参数说明如下：

| 名称              | 类型   | 说明                                           |
| ----------------- | ------ | ---------------------------------------------- |
| name              | string | 管道的名称，唯一不能重复                       |
| auto_start        | bool   | 是否随着网关自启动，也就是立即执行该任务       |
| keep_running      | bool   | 网关执行完毕之后是否继续重头开始执行           |
| singleton        | bool   | 该任务是否为单例，一个集群内只允许一个节点实例运行      |
| max_running_in_ms        | int   | 该任务运行执行的最大时间，默认 `60000` 毫秒     |
| retry_delay_in_ms | int    | 该任务再次执行的最少等待时间，默认 `5000` 毫秒 |
| processor         | array  | 该管道依次执行的处理器列表                     |

## 处理器列表

### 任务调度

- [dag](./dag)

### 消息处理

- [consumer](./consumer)
- [smtp](./smtp)
- [merge_to_bulk](./merge_to_bulk)
- [flow_replay](./flow_replay)

### 索引写入

- [bulk_indexing](./bulk_indexing)
- [json_indexing](./json_indexing)

### 索引对比

- [dump_hash](./dump_hash)
- [index_diff](./index_diff)


### 请求重放

- [replay](./replay)

