---
title: "clone"
asciinema: true
---

# clone

## 描述

clone 过滤器用来将流量克隆转发到另外的一个处理流程，可以实现双写、多写、多数据中心同步、集群升级、版本切换等需求。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: double_write
    filter:
      - clone:
          flows:
            - write_to_region_a
            - write_to_region_b #last one's response will be output to client
  - name: write_to_region_a
    filter:
      - elasticsearch:
          elasticsearch: es1
  - name: write_to_region_b
    filter:
      - elasticsearch:
          elasticsearch: es2
```

上面的例子可以将 Elasticsearch 的请求复制到两个不同的异地集群。

## 参数说明

| 名称     | 类型  | 说明                                                                                      |
| -------- | ----- | ----------------------------------------------------------------------------------------- |
| flows    | array | 指定多个流量处理的流程，依次同步执行，将最后一个流程处理的结果输出给客户端                |
| continue | bool  | 流量迁移出去之后，是否还继续执行之前的既定流程，设置成 `false` 则立即返回，默认 `false`。 |
