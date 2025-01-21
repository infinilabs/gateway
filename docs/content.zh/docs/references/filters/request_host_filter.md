---
title: "request_host_filter"
---

# request_host_filter

## 描述

request_host_filter 过滤器主要用来按照指定的域名或者主机名来进行请求过滤，适合只有一个 IP 多个域名需要进行域名访问控制的场景。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: test
    filter:
      - request_host_filter:
          include:
            - domain-test2.com:8000
```

上面的例子表示，只有访问的是这个域名 `domain-test2.com:8000` 的请求才被允许通过。

## 示例如下：

```
✗ curl -k -u user:passwd http://domain-test4.com:8000/   -v

*   Trying 192.168.3.67...
* TCP_NODELAY set
* Connected to domain-test4.com (192.168.3.67) port 8000 (#0)
* Server auth using Basic with user 'medcl'
> GET / HTTP/1.1
> Host: domain-test4.com:8000
> Authorization: Basic 123=
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 403 Forbidden
< Server: INFINI
< Date: Fri, 15 Jan 2021 13:53:01 GMT
< Content-Length: 0
< FILTERED: true
<
* Connection #0 to host domain-test4.com left intact
* Closing connection 0

✗ curl -k -u user:passwd http://domain-test2.com:8000/   -v

*   Trying 192.168.3.67...
* TCP_NODELAY set
* Connected to domain-test2.com (192.168.3.67) port 8000 (#0)
* Server auth using Basic with user 'medcl'
> GET / HTTP/1.1
> Host: domain-test2.com:8000
> Authorization: Basic 123=
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 200 OK
< Server: INFINI
< Date: Fri, 15 Jan 2021 13:52:53 GMT
< Content-Type: application/json; charset=UTF-8
< Content-Length: 480
< UPSTREAM: 192.168.3.203:9200
< CACHE-HASH: a2902f950b4ade804b21a062257387ef
<
{
  "name" : "node3",
  "cluster_name" : "pi",
  "cluster_uuid" : "Z_HcN_6ESKWicV-eLsyU4g",
  "version" : {
    "number" : "6.4.2",
    "build_flavor" : "default",
    "build_type" : "tar",
    "build_hash" : "04711c2",
    "build_date" : "2018-09-26T13:34:09.098244Z",
    "build_snapshot" : false,
    "lucene_version" : "7.4.0",
    "minimum_wire_compatibility_version" : "5.6.0",
    "minimum_index_compatibility_version" : "5.0.0"
  },
  "tagline" : "You Know, for Search"
}
* Connection #0 to host domain-test2.com left intact
* Closing connection 0
```

## 参数说明

| 名称    | 类型   | 说明                                                                        |
| ------- | ------ | --------------------------------------------------------------------------- |
| exclude | array  | 拒绝通过的请求的主机列表                                                    |
| include | array  | 允许通过的请求的主机列表                                                    |
| action  | string | 符合过滤条件之后的处理动作，可以是 `deny` 和 `redirect_flow`，默认为 `deny` |
| status  | int    | 自定义模式匹配之后返回的状态码                                              |
| message | string | 自定义 `deny` 模式返回的消息文本                                            |
| flow    | string | 自定义 `redirect_flow` 模式执行的 flow ID                                   |

{{< hint info >}}
注意: 当设置了 `include` 条件的情况下，必须至少满足 `include` 设置的其中一种响应码才能被允许通过。
当仅设置了 `exclude` 条件的情况下，不符合 `exclude` 的任意请求都允许通过。
{{< /hint >}}
