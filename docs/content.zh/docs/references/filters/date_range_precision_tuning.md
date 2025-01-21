---
title: "date_range_precision_tuning"
---

# date_range_precision_tuning

## 描述

date_range_precision_tuning 过滤器用来重设时间范围查询的时间精度，通过调整精度，可以让短时间内邻近的重复请求更容易被缓存，对于有一些对于时间精度不那么高但是数据量非常大的场景，比如使用 Kibana 来做报表分析，通过缩减精度来缓存重复的查询请求，从而降低后端服务器压力，前端报表展现的提速非常明显。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: test
    filter:
      - date_range_precision_tuning:
          time_precision: 4
      - get_cache:
      - elasticsearch:
          elasticsearch: dev
      - set_cache:
```

## 精度说明

Kibana 默认发往 Elasticsearch 的查询，使用的是当前时间 Now，精度到毫秒，通过设置不同的精度来改写查询，以下面的查询为例：

```
{"range":{"@timestamp":{"gte":"2019-09-26T08:21:12.152Z","lte":"2020-09-26T08:21:12.152Z","format":"strict_date_optional_time"}
```

分别设置不同的精度，改写之后的查询结果如下：

| 精度 | 新的查询                                                                                                                        |
| ---- | ------------------------------------------------------------------------------------------------------------------------------- |
| 0    | {"range":{"@timestamp":{"gte":"2019-09-26T00:00:00.000Z","lte":"2020-09-26T23:59:59.999Z","format":"strict_date_optional_time"} |
| 1    | {"range":{"@timestamp":{"gte":"2019-09-26T00:00:00.000Z","lte":"2020-09-26T09:59:59.999Z","format":"strict_date_optional_time"} |
| 2    | {"range":{"@timestamp":{"gte":"2019-09-26T08:00:00.000Z","lte":"2020-09-26T08:59:59.999Z","format":"strict_date_optional_time"} |
| 3    | {"range":{"@timestamp":{"gte":"2019-09-26T08:20:00.000Z","lte":"2020-09-26T08:29:59.999Z","format":"strict_date_optional_time"} |
| 4    | {"range":{"@timestamp":{"gte":"2019-09-26T08:21:00.000Z","lte":"2020-09-26T08:21:59.999Z","format":"strict_date_optional_time"} |
| 5    | {"range":{"@timestamp":{"gte":"2019-09-26T08:21:10.000Z","lte":"2020-09-26T08:21:19.999Z","format":"strict_date_optional_time"} |
| 6    | {"range":{"@timestamp":{"gte":"2019-09-26T08:21:12.000Z","lte":"2020-09-26T08:21:12.999Z","format":"strict_date_optional_time"} |
| 7    | {"range":{"@timestamp":{"gte":"2019-09-26T08:21:12.100Z","lte":"2020-09-26T08:21:12.199Z","format":"strict_date_optional_time"} |
| 8    | {"range":{"@timestamp":{"gte":"2019-09-26T08:21:12.150Z","lte":"2020-09-26T08:21:12.159Z","format":"strict_date_optional_time"} |
| 9    | {"range":{"@timestamp":{"gte":"2019-09-26T08:21:12.152Z","lte":"2020-09-26T08:21:12.152Z","format":"strict_date_optional_time"} |

## 参数说明

| 名称           | 类型  | 说明                                                                                                      |
| -------------- | ----- | --------------------------------------------------------------------------------------------------------- |
| time_precision | int   | 时间的精度长度，对于时间呈现长度位数，默认为 `4`，有效范围 0 到 9                                         |
| path_keywords  | array | 只对包含所设置关键字的请求进行时间精度重置，避免对不必要的请求进行解析，默认 `_search` 和 `_async_search` |
