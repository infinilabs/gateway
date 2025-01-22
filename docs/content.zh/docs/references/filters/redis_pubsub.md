---
title: "redis_pubsub"
---

# redis_pubsub

## 描述

reids 过滤器用来将收到的请求和响应结果保存到 Redis 消息队列中。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: redis_pubsub
    filter:
      - redis_pubsub:
          host: 127.0.0.1
          port: 6379
          channel: gateway
          response: true
```

## 参数说明

| 名称     | 类型   | 说明                                 |
| -------- | ------ | ------------------------------------ |
| host     | string | Reids 主机名，默认 `localhost`       |
| port     | int    | Reids 端口号，默认为 `6379`          |
| password | string | Redis 密码                           |
| db       | int    | Redis 默认选择的数据库，默认为 `0`   |
| channel  | string | Redis 消息队列名称，必填，没有默认值 |
| response | bool   | 是否包含响应结果，默认为 `true`      |
