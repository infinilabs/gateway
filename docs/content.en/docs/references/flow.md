---
title: "Handling Flow"
weight: 45
---

# Handling Flow

## Flow Definition

Requests received by each gateway are handled through a series of processes and then results are returned to the client. A process is called a `flow` in INFINI Gateway. See the following example.

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

The above example defines two flows: `hello_world` and `not_found`.
Each flow uses a filter named `echo` to output a string. A series of filters can be defined in each flow and they are executed in the defined sequence.

### Syntax Description

INFINI Gateway defines a flow in the stipulated format and supports flexible conditional parameters for logical judgment. The specific format is defined as follows:

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

In the format defined above, `filter_name` indicates the name of a filter, which is used to execute a specific task. `condition` below `when` is used to define specific conditional parameters for executing the task, and the filter task is skipped when the conditions are not met. In `parameters`, parameters related to the filter are set, and the parameters are separated by the line feed character.

## Conditional Judgment

Complex logical judgments can be defined in a flow of INFINI Gateway so that a filter can be executed only when certain conditions are met. See the following example.

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

### Parameter Description

| Name | Type  | Description                                                                                      |
| ---- | ----- | ------------------------------------------------------------------------------------------------ |
| then | array | A series of filters to be executed only when conditions defined in `condition` are met.          |
| else | array | A set of filters to be executed only when the conditions are not met. You do not have to set it. |

You can use `if` to make conditional judgment and logical selection in the case of multiple filters and use `when` to determine whether to execute a single filter.

## Condition Type

For various `condition` defined in a flow, you can use the current [request context](./context/) to judge whether a specific condition is met so as to achieve logical processing. The conditions support the combination of Boolean expressions (AND, NOT, and OR). The complete list of condition types is as follows:

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

The `equals` condition is used to judge whether the content of a field is the specified value. It is used for the exact match of characters and digits.

The following example determines whether the request method is of the GET type and `_ctx` is a specific keyword for accessing the request context:

```
equals:
  _ctx.request.method: GET
```

### contains

The `contains` condition is used to judge whether the content of a field contains a specific character value. Only support string field.

The following example judges whether the returned response body contains an error keyword:

```
contains:
  _ctx.response.body: "error"
```

### prefix

Use the `prefix` condition to determine whether the contents of a field begin with a specific character value, Only support string field.

The following example determines that the returned request path starts with a specific index name:

```
prefix:
  _ctx.request.path: "/filebeat"
```

### suffix

Use the `suffix` condition to determine whether the content of a field ends with a specific character value. Only support string field.

The following example determines whether the request is a search request:

```
suffix:
  _ctx.request.path: "/_search"
```

### regexp

The `regexp` condition is used to judge whether the content of a field meets the matching rules of a regular expression. Only support string field.

The following example judges whether the request URI is a query request:

```
regexp:
  _ctx.request.uri: ".*/_search"
```

### range

The `range` condition is used to judge whether the value of a field meets a specific range. It supports the `lt`, `lte`, `gt`, and `gte` types and only numeric fields are supported.

The following example judges the range of the status code:

```
range:
  _ctx.response.code:
    gte: 400
```

The following combination example judges the range of the response byte size:

```
range:
  _ctx.request.body_length.gte: 100
  _ctx.request.body_length.lt: 5000
```

### network

If the value of a field is an IP address, you can use the `network` condition to judge whether the field meets a specific network range, whether it supports standard IPv4 or IPv6, whether it supports the classless inter-domain routing (CIDR) expression, or whether it uses an alias in the following range:

| Name                      | Description                                                                                                                                                                                                          |
| ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| loopback                  | Matches the local loopback network address. Range: `127.0.0.0/8` or `::1/128`.                                                                                                                                       |
| unicast                   | Matches global unicast addresses defined in RFC 1122, RFC 4632, and RFC 4291, except the IPv4 broadcast address (255.255.255.255) but including private address ranges.                                              |
| multicast                 | Matches the broadcast address.                                                                                                                                                                                       |
| interface_local_multicast | Matches the local multicast address of an IPv6 interface.                                                                                                                                                            |
| link_local_unicast        | Matches the link-local unicast address.                                                                                                                                                                              |
| link_local_multicast      | Matches the link-local broadcast address.                                                                                                                                                                            |
| private                   | Matches the private address range defined in RFC 1918 (IPv4) and RFC 4193 (IPv6).                                                                                                                                    |
| public                    | Matches public addresses other than the local address, unspecified address, IPv4 broadcast address, link-local unicast address, link-local multicast address, interface local multicast address, or private address. |
| unspecified               | Matches an unspecified address (IPv4 address `0.0.0.0` or IPv6 address `::`).                                                                                                                                        |

The following example matches the local network address:

```
network:
  _ctx.request.client_ip: private
```

The following example specifies a subnet:

```
network:
  _ctx.request.client_ip: '192.168.3.0/24'
```

An array is supported and it is judged that the condition is met when any value in the array is met.

```
network:
  _ctx.request.client_ip: ['192.168.3.0/24', '10.1.0.0/8', loopback]
```

### exists

You can use the `exists` condition to judge whether a field exists. It supports the use of one or more character fields. See the following example:

```
exists: ['_ctx.request.user']
```

### in

You can use the `in` condition to judge whether a field has any value in a specified array. It supports a single field and the character and numeric types.

The following example judges the returned status code.

```
in:
  _ctx.response.status: [ 403,404,200,201 ]
```

### queue_has_lag

The `queue_has_lag` condition is used to judge whether one or more local disk queues are stacked with messages.

```
queue_has_lag: [ "prod", "prod-500" ]
```

When the queue type is FIFO, If you want to set the depth of a queue to be greater than a specified depth, add `>queue depth` to the end of the queue name. See the following example:

```
queue_has_lag: [ "prod>10", "prod-500>10" ]
```

The above example shows that the condition is met only when the queue depth exceeds `10`.

### consumer_has_lag

The `consumer_has_lag` condition is used to judge whether delay and message stacking occur in the consumer of a queue.

```
consumer_has_lag:
  queue: "primary-partial-success_bulk_requests"
  group: "my-group"
  name: "my-consumer-1"
```

### cluster_available

The `cluster_available` condition is used to judge the service availability of one or more Elasticsearch clusters. See the following example:

```
cluster_available: ["prod"]
```

### or

The `or` condition is used to combine multiple optional conditions in the following format:

```
or:
  - <condition1>
  - <condition2>
  - <condition3>
  ...
```

See the following example:

```
or:
  - equals:
      _ctx.response.code: 304
  - equals:
      _ctx.response.code: 404
```

### and

The `and` condition is used to combine multiple necessary conditions in the following format:

```
and:
  - <condition1>
  - <condition2>
  - <condition3>
  ...
```

See the following example:

```
and:
  - equals:
      _ctx.response.code: 200
  - equals:
      _ctx.status: OK
```

You can combine the `and` and `or` conditions flexibly. See the following example:

```
or:
  - <condition1>
  - and:
    - <condition2>
    - <condition3>
```

### not

If you want to negate a condition, use the `not` condition in the following format:

```
not:
  <condition>
```

See the following example:

```
not:
  equals:
    _ctx.status: OK
```
