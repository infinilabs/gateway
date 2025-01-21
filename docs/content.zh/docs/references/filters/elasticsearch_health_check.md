---
title: "elasticsearch_health_check"
---

# elasticsearch_health_check

## 描述

elasticsearch_health_check 过滤器用来以限速模式下主动探测 Elasticsearch 的健康情况，
当出现后端故障的情况下，可以触发一次主动的集群健康检查，而不用等待 Elasticsearch 默认的轮询检查结果，限速设置为最多每秒发送一次检查请求给后端 Elasticsearch。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: elasticsearch_health_check
    filter:
      - elasticsearch_health_check:
          elasticsearch: dev
```

## 参数说明

| 名称          | 类型   | 说明                                         |
| ------------- | ------ | -------------------------------------------- |
| elasticsearch | string | 集群 ID                                      |
| interval      | int    | 设置最少执行请求的时间间隔，单位秒，默认 `1` |
