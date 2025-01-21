---
title: "queue_consumer"
---

# queue_consumer

## 描述

queue_consumer 处理器用来异步消费队列里面的请求到 Elasticsearch。

## 配置示例

一个简单的示例如下：

```
pipeline:
- name: bulk_request_ingest
  auto_start: true
  keep_running: true
  processor:
    - queue_consumer:
        input_queue: "backup"
        elasticsearch: "backup"
        waiting_after: [ "backup_failure_requests"]
        worker_size: 20
        when:
          cluster_available: [ "backup" ]
```

## 参数说明

| 名称                    | 类型   | 说明                                                                              |
| ----------------------- | ------ | --------------------------------------------------------------------------------- |
| input_queue             | string | 订阅的队列名称                                                                    |
| worker_size             | int    | 并行执行消费任务的线程数，默认 `1`                                                |
| idle_timeout_in_seconds | int    | 消费队列的超时时间，默认 `1`                                                      |
| elasticsearch           | string | 保存到目标集群的名称                                                              |
| waiting_after           | array  | 需要先等将这些指定队列消费完才能开始消费主队列里面的数据                          |
| failure_queue           | string | 因为后端故障执行失败的请求，默认为 `%input_queue%-failure`                        |
| invalid_queue           | string | 状态码返回为 4xx 的请求，默认为 `%input_queue%-invalid`                           |
| compress                | bool   | 是否压缩请求，默认 `false`                                                        |
| safety_parse            | bool   | 是否启用安全解析，即不采用 buffer 的方式，占用内存更高一点，默认为 `true`         |
| doc_buffer_size         | bool   | 单次请求处理的最大文档 buff size，建议设置超过单个文档的最大大小，默认 `256*1024` |
