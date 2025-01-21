---
title: "处理流程"
weight: 45
---

# 处理流程

## 流程定义

每一个网关接收到的请求都会通过一系列的流程处理，最后才返回给客户端，流程的定义在极限网关里面叫做 `flow`，以下面的这个例子为例：

```
flow:
  - name: hello_world
    filter:
      - echo:
          message: "hello gateway\n"
          repeat: 1
  - name: not_found
    filter:
      - echo:
          message: '404 not found\n'
          repeat: 1
```

上面的例子定义了两个 flow `hello_world` 和 `not_found`，
每个 flow 都使用了一个名为 `echo` 的过滤器，用来输出一段字符串，每个 flow 下面可以定义一系列 filter，他们按照定义的顺序依次执行。

### 语法说明

极限网关采用约定的格式来定义流程，并且支持灵活的条件参数来进行逻辑判断，具体的格式定义如下：

```
flow:
  - name: <flow_name>
    filter:
      - <filter_name>:
          when:
            <condition>
          <parameters>
      - <filter_name>:
          when:
            <condition>
          <parameters>
    ...
```

上面的 `filter_name` 代表具体的某个过滤器名称，用来执行特定的任务，`when` 下面的 `condition` 用来定义特定的满足执行该任务的条件参数，不满足条件的情况下会跳过该过滤器任务的执行，`parameters` 里面设置的该过滤器相关的参数，如果多个参数依次换行即可。

## 条件判断

极限网关的流程定义支持复杂的逻辑判断，可以让特定的过滤器只有在满足某种条件下才会执行，举例如下：

```
filter:
  - if:
      <condition>
    then:
      - <filter_name>:
          <parameters>
      - <filter_name>:
          <parameters>
      ...
    else:
      - <filter_name>:
          <parameters>
      - <filter_name>:
          <parameters>
      ...
```

### 参数说明

| 名称 | 类型  | 说明                                                      |
| ---- | ----- | --------------------------------------------------------- |
| then | array | 表示满足 `condition` 条件定义后才会执行的一系列过滤器定义 |
| else | array | 不满足条件才会执行的一系列过滤器定义，可不设置            |

使用 `if` 可以对多个 filter 来进行条件判断进行逻辑选择，使用 `when` 来对单个过滤器进行判断是否执行。

## 条件类型

在流程里面定义的各种 `condition` 条件可以使用当前[请求上下文](./context/) 来判断是否满足特定条件，从而实现逻辑处理，支持布尔表达式（AND、NOT、OR）来进行组合，完整的条件类型如下：

- equals
- contains
- prefix
- suffix
- regexp
- range
- network
- exists
- in
- queue_has_lag
- consumer_has_lag
- cluster_available
- or
- and
- not

### equals

使用 `equals` 条件来判断字段的内容是否为指定的值，用于字符和数字类型的精确匹配。

如下面的例子判断是否请求的方法是否为 GET 类型，`_ctx` 是访问请求上下文的特定关键字：

```
equals:
  _ctx.request.method: GET
```

### contains

使用 `contains` 条件来判断字段的内容是否包含特定的字符值，仅支持字符字段类型。

如下面的例子为判断返回的请求体里面是否包含错误关键字：

```
contains:
  _ctx.response.body: "error"
```

### prefix

使用 `prefix` 条件来判断字段的内容是否由特定的字符值开头，仅支持字符字段类型。

如下面的例子为判断返回的请求路径为特定索引名称开头：

```
prefix:
  _ctx.request.path: "/filebeat"
```

### suffix

使用 `suffix` 条件来判断字段的内容是否由特定的字符值结尾，仅支持字符字段类型。

如下面的例子为判断返回的请求是否为搜索请求：

```
suffix:
  _ctx.request.path: "/_search"
```

### regexp

使用 `regexp` 条件可以用来判断某个字段的内容是否满足正则表达式的匹配规则，仅支持字符字段类型。

如下面的例子判断请求的 uri 是否为查询请求：

```
regexp:
  _ctx.request.uri: ".*/_search"
```

### range

使用 `range` 条件用来判断字段的值是否满足特定的范围，支持 `lt`、`lte`、`gt` 和 `gte` 几种类型，仅支持数字字段类型。

如下面判断状态码范围的例子：

```
range:
  _ctx.response.code:
    gte: 400
```

以及如下组合来判断响应字节大小范围的例子：

```
range:
  _ctx.request.body_length.gte: 100
  _ctx.request.body_length.lt: 5000
```

### network

如果某个字段的值为 IP 字段类型，可以使用 `network` 条件可以判断该字段是否满足某个特定的网络范围，支持标准的 IPv4 和 IPv6，支持 CIDR 的表达方式，或者是以下范围别名：

| 名称                      | 说明                                                                                                                  |
| ------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| loopback                  | 匹配本地回环网络地址，范围：`127.0.0.0/8` 或者 `::1/128`。                                                            |
| unicast                   | 匹配 RFC 1122、RFC 4632 和 RFC 4291 中定义的全球单播地址，但 IPv4 广播地址 (255.255.255.255) 除外。包括私有地址范围。 |
| multicast                 | 匹配广播地址。                                                                                                        |
| interface_local_multicast | 匹配 IPv6 接口本地组播地址。                                                                                          |
| link_local_unicast        | 匹配链路本地单播地址。                                                                                                |
| link_local_multicast      | 匹配链路本地广播地址。                                                                                                |
| private                   | 匹配 RFC 1918 (IPv4) 和 RFC 4193 (IPv6) 中定义的私有地址范围。                                                        |
| public                    | 匹配除了本机、未指定、IPv4 广播、链路本地单播、链路本地多播、接口本地多播或私有地址以外的公网地址。                   |
| unspecified               | 匹配未指定的地址（IPv4 地址 `0.0.0.0` 或 IPv6 地址 `::` ）。                                                          |

如下面的例子匹配本机网络地址：

```
network:
  _ctx.request.client_ip: private
```

或者指定一个子网：

```
network:
  _ctx.request.client_ip: '192.168.3.0/24'
```

支持数组，任意满足即可：

```
network:
  _ctx.request.client_ip: ['192.168.3.0/24', '10.1.0.0/8', loopback]
```

### exists

如果要判断某个字段是否存在，可以使用 `exists`，支持一个或者多个字符字段，如下：

```
exists: ['_ctx.request.user']
```

### in

如果要判断某个字段是否存在指定数组的任意值，可以使用 `in`，支持单个字段的判断，仅支持字符和数值类型。

如下判断返回状态码：

```
in:
  _ctx.response.status: [ 403,404,200,201 ]
```

### queue_has_lag

使用 `queue_has_lag` 可以来判断某个或多个本地磁盘队列是否存在堆积的情况，如下：

```
queue_has_lag: [ "prod", "prod-500" ]
```

当队列类型为 FIFO 时，如果希望设置队列大于指定深度可以在队列的名称后面加上 `>队列深度`，如：

```
queue_has_lag: [ "prod>10", "prod-500>10" ]
```

上面的例子表示，只有当队列深度超过 `10` 的情况下才满足条件。

### consumer_has_lag

使用 `consumer_has_lag` 可以来判断某个队列的消费者是否存在延迟堆积的情况，如下：

```
consumer_has_lag:
  queue: "primary-partial-success_bulk_requests"
  group: "my-group"
  name: "my-consumer-1"
```

### cluster_available

使用 `cluster_available` 可以判断某个或多个 Elasticsearch 集群的服务可用性，如下：

```
cluster_available: ["prod"]
```

### or

使用 `or` 来组合多个任意可选条件，格式如下：

```
or:
  - <condition1>
  - <condition2>
  - <condition3>
  ...
```

举例如下：

```
or:
  - equals:
      _ctx.response.code: 304
  - equals:
      _ctx.response.code: 404
```

### and

使用 `and` 来组合多个必要条件，格式如下：

```
and:
  - <condition1>
  - <condition2>
  - <condition3>
  ...
```

举例如下：

```
and:
  - equals:
      _ctx.response.code: 200
  - equals:
      _ctx.status: OK
```

还可以对 `and` 和 `or` 条件进行灵活组合，如下：

```
or:
  - <condition1>
  - and:
    - <condition2>
    - <condition3>
```

### not

如果要对某个条件取反，使用 `not` 即可，格式如下：

```
not:
  <condition>
```

举例如下：

```
not:
  equals:
    _ctx.status: OK
```
