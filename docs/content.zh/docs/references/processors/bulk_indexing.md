---
title: "bulk_indexing"
---

# bulk_indexing

## 描述

bulk_indexing 处理器用来异步消费队列里面的 bulk 请求。

## 配置示例

一个简单的示例如下：

```
pipeline:
- name: bulk_request_ingest
  auto_start: true
  keep_running: true
  processor:
    - bulk_indexing:
        queue_selector.labels:
          type: bulk_reshuffle
          level: cluster
```

## 参数说明

| 名称                                               | 类型     | 说明                                                                                                                               |
| -------------------------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| elasticsearch                                      | string   | 默认的 Elasticsearch 集群 ID,如果队列 Labels 里面没有指定 `elasticsearch` 的话会使用这个参数                                       |
| max_connection_per_node                            | int      | 目标节点允许的最大连接数，默认 `1`                                                                                                 |
| max_worker_size                                    | int      | 最大允许同时运行的 worker 大小,默认 `10`                                                                                           |
| num_of_slices                                      | int      | 并行消费单个队列的线程, 运行时最大的 slice 大小                                                                                    |
| slices                                             | array    | 允许的 slice 编号, int 数组                                                                                                        |
| queue_selector.labels                              | map      | 根据 Label 来过滤一组需要消费的队列, 同 `queues` 配置                                                                              |
| queue_selector.ids                                 | array    | 指定要消费的队列的 UUID, 字符数组                                                                                                  |
| queue_selector.keys                                | array    | 指定要消费的队列的唯一 Key 路径, 字符数组                                                                                          |
| queues                                             | map      | 根据 Label 来过滤一组需要消费的队列, 同 `queue_selector.labels` 配置                                                               |
| waiting_after                                      | array    | 是否等待指定队列消费完成才开始消费, 队列的 UUID, 字符数组                                                                          |
| idle_timeout_in_seconds                            | int      | 消费队列的超时时间，默认 `5`, 即 5s                                                                                                |
| detect_active_queue                                | bool     | 是否自动检测符合条件的新的队列,默认 `true`                                                                                         |
| detect_interval                                    | bool     | 自动检测符合条件的新的队列的时间间隔,单位毫秒, 默认 `5000`                                                                         |
| skip_info_missing                                  | bool     | 忽略不满足条件的队列，如节点、索引、分片信息不存在时则需等待信息获取后再消费，默认为 `false`，否则会随机挑选一个 es 节点来发送请求 |
| skip_empty_queue                                   | bool     | 是否跳过空队列的消费, 默认 `true`                                                                                                  |
| consumer.source                                    | string   | 消费者来源                                                                                                                         |
| consumer.id                                        | string   | 消费者唯一标识                                                                                                                     |
| consumer.name                                      | string   | 消费者名称                                                                                                                         |
| consumer.group                                     | string   | 消费者组名称                                                                                                                       |
| consumer.fetch_min_bytes                           | int      | 拉取消息最小的字节大小, 默认 `1`                                                                                                   |
| consumer.fetch_max_bytes                           | int      | 拉取消息最大的字节大小, 默认 `10485760`, 即 10MB                                                                                   |
| consumer.fetch_max_messages                        | int      | 拉取最大的消息个数, 默认 `1`                                                                                                       |
| consumer.fetch_max_wait_ms                         | int      | 拉取最大的等待时间, 单位毫秒, 默认 `10000`                                                                                         |
| consumer.eof_retry_delay_in_ms                     | int      | 达到文件末尾重试的等待时间, 单位毫秒, 默认 `500`                                                                                   |
| bulk.compress                                      | bool     | 是否开启请求压缩                                                                                                                   |
| bulk.batch_size_in_kb                              | int      | 批次请求的单位大小，单位 `KB`                                                                                                      |
| bulk.batch_size_in_mb                              | int      | 批次请求的单位大小，单位 `MB`,默认 `10`                                                                                            |
| bulk.batch_size_in_docs                            | int      | 批次请求的文档个数, 默认 `1000`                                                                                                    |
| bulk.retry_delay_in_seconds                        | int      | 请求重试的等待时间，默认 `1`                                                                                                       |
| bulk.reject_retry_delay_in_seconds                 | int      | 请求拒绝的等待时间，默认 `1`                                                                                                       |
| bulk.max_retry_times                               | int      | 最大重试次数                                                                                                                       |
| bulk.request_timeout_in_second                     | int      | HTTP 请求执行的超时时间                                                                                                            |
| bulk.invalid_queue                                 | string   | 因为请求不合法的 4xx 请求队列                                                                                                      |
| bulk.dead_letter_queue                             | string   | 超过最大重试次数的请求队列                                                                                                         |
| bulk.remove_duplicated_newlines                    | bool     | 是否主动移除 Bulk 请求里面重复的换行符                                                                                             |
| bulk.response_handle.save_success_results          | bool     | 是否保存执行成功的请求结果，默认 `false`                                                                                           |
| bulk.response_handle.output_bulk_stats             | bool     | 输出 bulk 统计信息，默认 `false`                                                                                                   |
| bulk.response_handle.include_index_stats           | bool     | 将索引信息包含在 bulk 统计信息内，默认 `true`                                                                                      |
| bulk.response_handle.include_action_stats          | bool     | 将索引信息包含在 bulk 统计信息内，默认 `true`                                                                                      |
| bulk.response_handle.save_error_results            | bool     | 是否保存执行出错的请求结果，默认 `true`                                                                                            |
| bulk.response_handle.include_error_details         | bool     | 包含额外的单条请求的错误日志，默认 `true`                                                                                          |
| bulk.response_handle.max_error_details_count       | bool     | 单条请求的错误日志总条数，默认 `50`                                                                                                |
| bulk.response_handle.save_busy_results             | bool     | 是否保存繁忙 429 的日志，默认 `true`                                                                                               |
| bulk.response_handle.bulk_result_message_queue     | string   | 保存异步日志的消息队列名称，默认 `bulk_result_messages`                                                                            |
| bulk.response_handle.max_request_body_size         | int      | 最大的请求体大小，超出截断，默认 10k 即 `10240`                                                                                    |
| bulk.response_handle.max_response_body_size        | int      | 最大的响应体大小，超出截断，默认 10k 即 `10240`                                                                                    |
| bulk.response_handle.retry_rules.retry_429         | bool     | 是否重试 429，默认 `true`                                                                                                          |
| bulk.response_handle.retry_rules.retry_4xx         | bool     | 是否重试 429 以外的 4xx 状态码 ，默认 `false`                                                                                      |
| bulk.response_handle.retry_rules.default           | bool     | 是否重试`retry_rules`未配置的其他状态码，默认`true`                                                                                |
| bulk.response_handle.retry_rules.permitted.status  | []int    | 允许重试的状态码列表                                                                                                               |
| bulk.response_handle.retry_rules.permitted.keyword | []string | 允许重试的关键字列表，只要是请求里面包含该任意关键字则重试                                                                         |
| bulk.response_handle.retry_rules.denied.status     | []int    | 不允许重试的状态码列表                                                                                                             |
| bulk.response_handle.retry_rules.denied.keyword    | []string | 不允许重试的关键字列表，只要是请求里面包含该任意关键字则不重试                                                                     |
