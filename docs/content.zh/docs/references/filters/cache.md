---
title: "cache"
asciinema: true
---

# cache

## 描述

cache 过滤器由 `get_cache` 和 `set_cache` 两组过滤器组成，一般需要组合使用，可用于缓存加速查询，抵挡重复请求，降低后端集群查询压力。

## get_cache 过滤器

过滤器 `get_cache` 用来从缓存里面获取之前出现的消息，直接返回给客户端，避免访问后端 Elasticsearch，用于缓存热点数据。

配置示例如下：

```
flow:
  - name: get_cache
    filter:
      - get_cache:
          pass_patterns: ["_cat","scroll", "scroll_id","_refresh","_cluster","_ccr","_count","_flush","_ilm","_ingest","_license","_migration","_ml","_rollup","_data_stream","_open", "_close"]
```

### 参数说明

| 名称          | 类型   | 说明                                                       |
| ------------- | ------ | ---------------------------------------------------------- |
| pass_patterns | string | 设置忽略缓存的请求规则，URL 包含其中的任意关键字将跳过缓存 |

## set_cache 过滤器

过滤器 `set_cache` 用来将后端查询拿到的返回结果存到缓存里面，可以设置过期时间。

配置示例如下：

```
flow:
  - name: get_cache
    filter:
      - set_cache:
          min_response_size: 100
          max_response_size: 1024000
          cache_ttl: 30s
          max_cache_items: 100000
```

### 参数说明

| 名称                   | 类型   | 说明                                                                    |
| ---------------------- | ------ | ----------------------------------------------------------------------- |
| cache_type             | string | 缓存类型，支持 `ristretto`，`ccache` 和 `redis`，默认 `ristretto`       |
| cache_ttl              | string | 缓存的过期时间，默认 `10s`                                              |
| async_search_cache_ttl | string | 异步请求结果的缓存过期时间，默认 `10m`                                  |
| min_response_size      | int    | 最小符合缓存要求的消息体大小，默认 `-1` 表示不限制                      |
| max_response_size      | int    | 最大符合缓存要求的消息体大小，默认为 int 的最大值                       |
| max_cached_item        | int    | 最大的缓存消息总数，默认 `1000000`，当类型为 `ccache`有效               |
| max_cached_size        | int    | 最大的缓存内存开销，默认 `1000000000` 即 1GB，当类型为 `ristretto` 有效 |
| validated_status_code  | array  | 允许被缓存的请求状态码，默认 `200,201,404,403,413,400,301`              |

## 其它参数

如果希望主动忽略缓存，可以在 URL 的参数里面传递一个 `no_cache` 来让网关忽略缓存。如：

```
curl http://localhost:8000/_search?no_cache=true
```
