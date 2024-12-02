---
title: "cache"
asciinema: true
---

# cache

## Description

The cache filter is composed of the `get_cache` and `set_cache` filters, which need to be used in combination. The cache filter is used to cache accelerated queries, prevent repeated requests, and reduce the query pressure of back-end clusters.

## get_cache Filter

The `get_cache` filter is used to acquire previous messages from the cache and return them to the client, without needing to access the back-end Elasticsearch. It is intended to cache hotspot data.

A configuration example is as follows:

```
flow:
  - name: get_cache
    filter:
      - get_cache:
          pass_patterns: ["_cat","scroll", "scroll_id","_refresh","_cluster","_ccr","_count","_flush","_ilm","_ingest","_license","_migration","_ml","_rollup","_data_stream","_open", "_close"]
```

### Parameter Description

| Name          | Type   | Description                                                                                                |
| ------------- | ------ | ---------------------------------------------------------------------------------------------------------- |
| pass_patterns | string | Rule for ignoring the cache for a request. The cache is skipped when the URL contains any defined keyword. |

## set_cache Filter

The `set_cache` filter is used to cache results returned through back-end query. Expiration time can be set for the cache.

A configuration example is as follows:

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

### Parameter Description

| Name                   | Type   | Description                                                                                                                             |
| ---------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------- |
| cache_type             | string | Cache type. It can be set to `ristretto`, `ccache`, or `redis`, and the default value is `ristretto`.                                   |
| cache_ttl              | string | Expiration time of the cache. The default value is `10s`.                                                                               |
| async_search_cache_ttl | string | Expiration time of the cache for storing asynchronous request results. The default value is `10m`.                                      |
| min_response_size      | int    | Minimum message body size that meets cache requirements. The default value is `-1`, indicating an unlimited value.                      |
| max_response_size      | int    | Maximum message body size that meets cache requirements. The default value is the maximum value of the int parameter.                   |
| max_cached_item        | int    | Maximum number of messages that can be cached. The default value is `1000000`. The value is valid when the cache type is `ccache`.      |
| max_cached_size        | int    | Maximum cache memory overhead. The default value is `1000000000`, that is, 1 GB. The value is valid when the cache type is `ristretto`. |
| validated_status_code  | array  | Request status code that is allowed to be cached. The default value is `200,201,404,403,413,400,301`.                                   |

## Other Parameters

If you want to ignore caching, you can define `no_cache` in the URL parameters to cause the gateway to ignore caching. For example:

```
curl http://localhost:8000/_search?no_cache=true
```
