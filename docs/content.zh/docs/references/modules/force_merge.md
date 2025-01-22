---
title: "索引段合并"
weight: 30
---

# 主动合并索引分段

极限网关内置一个索引分段合并服务，可以主动对索引段文件进行合并，从而提升查询速度，段合并服务支持多个索引的依次顺序处理，并对合并任务状态进行了跟踪处理，避免大量段合并任务并行操作拖慢集群。

## 如何开启

修改配置文件 `gateway.yml`，增加如下配置：

```
force_merge:
  enabled: false
  elasticsearch: dev
  min_num_segments: 20
  max_num_segments: 1
  indices:
    - index_name
```

各参数说明如下：

| 名称                             | 类型   | 说明                                                           |
| -------------------------------- | ------ | -------------------------------------------------------------- |
| enabled                          | bool   | 是否启用该模块，默认是 `false`                                 |
| elasticsearch                    | string | 操作的 Elasticsearch 集群 ID                                   |
| min_num_segments                 | int    | 超过多少分片的索引才会执行主动分片合并，以索引为单位的统计数目 |
| max_num_segments                 | int    | 将分片下的段文件合并之后，最多生成的段文件个数                 |
| indices                          | array  | 需要进行分片合并的索引列表                                     |
| discovery                        | object | 自动发现索引的相关设置                                         |
| discovery.min_idle_time          | string | 满足段合并条件的最小时间跨度，默认 `1d`                        |
| discovery.interval               | string | 重新检测需要进行段合并的时间间隔                               |
| discovery.rules                  | array  | 自动进行索引检测的索引匹配规则                                 |
| discovery.rules.index_pattern    | string | 要进行索引段文件合并的索引通配符                               |
| discovery.rules.timestamp_fields | array  | 代表索引时间戳的字段列表                                       |
