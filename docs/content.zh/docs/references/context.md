---
title: "请求上下文"
weight: 49
---

# 请求上下文

## 什么是上下文

上下文是极限网关用来访问当前运行环境下相关信息的入口，如请求的来源和配置信息等等，使用关键字 `_ctx` 即可访问相应的字段，如：`_ctx.request.uri` 表示请求的 URL 地址。

## 内置请求上下文

HTTP 请求内置的 `_ctx` 上下文对象主要包括如下：

| 名称        | 类型   | 说明                     |
| ----------- | ------ | ------------------------ |
| id          | uint64 | 请求的唯一 ID            |
| tls         | bool   | 表示请求是否 TLS         |
| remote_ip   | string | 客户端来源 IP            |
| remote_addr | string | 客户端来源地址，包含端口 |
| local_ip    | string | 网关本地 IP              |
| local_addr  | string | 网关本地地址，包含端口   |
| elapsed     | int64  | 请求已执行时间（毫秒）   |
| request.\*  | object | 描述请求信息             |
| response.\* | object | 描述响应信息             |

### request

`request` 对象包含以下属性：

| 名称        | 类型   | 说明                         |
| ----------- | ------ | ---------------------------- |
| to_string   | string | 文本格式的 HTTP 完整请求信息 |
| host        | string | 访问的目标主机名/域名        |
| method      | string | 请求类型                     |
| uri         | string | 请求完整地址                 |
| path        | string | 请求路径                     |
| query_args  | map    | Url 请求参数                 |
| username    | string | 发起请求的用户名             |
| password    | string | 发起请求的密码信息           |
| header      | map    | Header 参数                  |
| body        | string | 请求体                       |
| body_json   | object | JSON 请求体对象              |
| body_length | int    | 请求体长度                   |

如果客户端提交的请求体数据类型是 JSON 格式，可以通过 `body_json` 来直接访问，举例如下：

```
curl -u tesla:password -XGET "http://localhost:8000/medcl/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query":{
	"bool":{
	"must":[{"match":{"name":"A"}},{"match":{"age":18}}]
	}
},
"size":900,
  "aggs": {
    "total_num": {
      "terms": {
        "field": "name1",
        "size": 1000000
      }
    }
  }
}'
```

在 JSON 里面通过 `.` 来标识路径，如果是数组则使用 `[下标]` 来访问指定的元素，比如可以使用一个 `dump` 过滤器来进行调试，如下：

```
  - name: cache_first
    filter:
      - dump:
          context:
            - _ctx.request.body_json.size
            - _ctx.request.body_json.aggs.total_num.terms.field
            - _ctx.request.body_json.query.bool.must.[1].match.age
```

输出结果如下：

```
_ctx.request.body_json.size  :  900
_ctx.request.body_json.aggs.total_num.terms.field  :  name1
_ctx.request.body_json.query.bool.must.[1].match.age  :  18
```

### response

`response` 对象包含以下属性：

| 名称         | 类型   | 说明                         |
| ------------ | ------ | ---------------------------- |
| to_string    | string | 文本格式的 HTTP 完整响应信息 |
| status       | int    | 请求状态码                   |
| header       | map    | Header 参数                  |
| content_type | string | 响应请求体类型               |
| body         | string | 响应体                       |
| body_json    | object | JSON 请求体对象              |
| body_length  | int    | 响应体长度                   |

## 系统上下文

系统上下文对象 `_sys.*` 有如下属性：

| 名称           | 类型   | 说明                                        |
| -------------- | ------ | ------------------------------------------- |
| hostname       | string | 网关所在服务器主机名                        |
| month_of_now   | int    | 当前时间的月份，范围 `[1,12]`               |
| weekday_of_now | int    | 当前时间的周几，范围 `[0,6]`, `0` is Sunday |
| day_of_now     | int    | 当前时间的自然天值                          |
| hour_of_now    | int    | 当前时间的小时值，范围 `[0,23]`             |
| minute_of_now  | int    | 当前时间的分钟值，范围 `[0,59]`             |
| second_of_now  | int    | 当前时间的秒值，范围 `[0,59]`               |
| unix_timestamp_of_now  | int    | 当前时间的 Unix 时间戳           |
| unix_timestamp_milli_of_now  | int64    | 当前时间的 Unix 时间戳，毫秒精度 |

## 其它

 `_util.*` 主要用于获取一些特殊的变量:

| 名称           | 类型   | 说明                                        |
| --------------- | ------- | ------------------------------------------- |
| generate_uuid   | string  | 生成一个随机 UUID   |
| increment_id    | string  | 生成一个自增 ID，默认桶名 `default`, 支持自定义, e.g., `_util.increment_id.mybucket` |