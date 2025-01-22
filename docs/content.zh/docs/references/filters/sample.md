---
title: "sample"
asciinema: true
---

# sample

## 描述

sample 过滤器用来将正常的流量按照比例采样，对于海量查询的场景，全流量收集日志需要耗费大量的资源，可以考虑进行抽样统计，对查询日志进行采样分析。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: sample
    filter:
      - sample:
          ratio: 0.2
```

## 参数说明

| 名称  | 类型  | 说明     |
| ----- | ----- | -------- |
| ratio | float | 采样比例 |
