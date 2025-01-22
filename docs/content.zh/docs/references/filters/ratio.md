---
title: "ratio"
asciinema: true
---

# ratio

## 描述

ratio 过滤器用来将正常的流量按照比例迁移转发到另外的一个处理流程，可以实现灰度发布、流量迁移导出，或者将部分流量切换到不同版本集群用于测试的能力。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: ratio_traffic_forward
    filter:
      - ratio:
          ratio: 0.1
          flow: hello_world
          continue: true
```

## 参数说明

| 名称     | 类型   | 说明                                                                                      |
| -------- | ------ | ----------------------------------------------------------------------------------------- |
| ratio    | float  | 需要迁移的流量比例                                                                        |
| action   | string | 当命中之后的行为，可以为 `drop` 或 `redirect_flow`，默认 `redirect_flow`                  |
| flow     | string | 指定新的流量处理流程                                                                      |
| continue | bool   | 流量迁移出去之后，是否还继续执行之前的既定流程，设置成 `false` 则立即返回，默认 `false`。 |
