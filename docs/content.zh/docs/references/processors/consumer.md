---
title: "consumer"
---

# consumer

## 描述

consumer 处理器用来消费 `queue` 记录的消息请求，但是不处理，目标是提供数据消费管道的入口，由后续的 processor 进行数据加工。

## 配置示例

一个简单的示例如下：

```
pipeline:
  - name: consume_queue_messages
    auto_start: true
    keep_running: true
    retry_delay_in_ms: 5000
    processor:
      - consumer:
          consumer:
            fetch_max_messages: 1
          max_worker_size: 200
          num_of_slices: 1
          idle_timeout_in_seconds: 30
          queue_selector:
            keys:
              - email_messages
          processor:
            - xxx1:
            - xxx2:
```

上面的例子，订阅并消费队列 `email_messages`，队列消息保存在当前 Pipeline 管道的上下文里面，Consumer 提供了一个 `processor` 参数，这个参数里面是一系列 Processor，依次执行，任何一个 Processor 如果执行返回出错，`consumer` 则退出切不会 commit 这批数据。

## 参数说明

| 名称     | 类型   | 说明                                   |
| -------- | ------ | -------------------------------------- |
| message_field                                    | string      |  从队列获取到的消息，存放到上下文的字段名称, 默认 `messages`                                                                                           |
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
| detect_interval                                    | int     | 自动检测符合条件的新的队列的时间间隔,单位毫秒, 默认 `5000`                                                                         |
| quite_detect_after_idle_in_ms                      | bool     | 退出自动检测的闲置时间间隔,单位毫秒, 默认 `30000`                                                                         |
| skip_empty_queue                                   | bool     | 是否跳过空队列的消费, 默认 `true`                                                                                                  |
| quit_on_eof_queue                                   | bool     | 队列执行到最后一条消息自动退出消费, 默认 `true`                                                                                                  |
| consumer.source                                    | string   | 消费者来源                                                                                                                         |
| consumer.id                                        | string   | 消费者唯一标识                                                                                                                     |
| consumer.name                                      | string   | 消费者名称                                                                                                                         |
| consumer.group                                     | string   | 消费者组名称                                                                                                                       |
| consumer.fetch_min_bytes                           | int      | 拉取消息最小的字节大小, 默认 `1`                                                                                                   |
| consumer.fetch_max_bytes                           | int      | 拉取消息最大的字节大小, 默认 `10485760`, 即 10MB                                                                                   |
| consumer.fetch_max_messages                        | int      | 拉取最大的消息个数, 默认 `1`                                                                                                       |
| consumer.fetch_max_wait_ms                         | int      | 拉取最大的等待时间, 单位毫秒, 默认 `10000`                                                                                         |
| consumer.eof_retry_delay_in_ms                     | int      | 达到文件末尾重试的等待时间, 单位毫秒, 默认 `500`                                                                                   |
