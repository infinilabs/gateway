---
title: "服务路由"
weight: 30
---

# 服务路由

极限网关通过路由来判断流量的去向，一个典型的路由配置示例如下：

```
router:
  - name: my_router
    default_flow: default_flow
    tracing_flow: request_logging
    rules:
      - method:
          - PUT
          - POST
        pattern:
          - "/_bulk"
          - "/{index_name}/_bulk"
        flow:
          - bulk_process_flow
```

路由有几个非常重要的概念：

- flow：请求的处理流程，一个路由里面有三个地方定义 flow
- default_flow: 默认的处理流，也就是业务处理的主流程，请求转发、过滤、缓存等操作都在这里面进行
- tracing_flow：用于追踪请求状态的流，不受 default_flow 的影响，用于记录请求日志、统计等
- rules：根据匹配规则将请求分发到特定的处理流中去，支持请求的 Method、Path 的正则匹配

## 参数说明

| 名称                     | 类型         | 说明                                                                                                                       |
| ------------------------ | ------------ | -------------------------------------------------------------------------------------------------------------------------- |
| name                     | string       | 路由名称                                                                                                                   |
| default_flow             | string       | 默认的请求的处理流程名称                                                                                                   |
| tracing_flow             | string       | 用于追踪请求的处理流程名称                                                                                                 |
| rules                    | array        | 路由规则列表，按照数组的先后顺序依次应用                                                                                   |
| rules.method             | string       | 请求的 Method 类型，支持 `GET`、`HEAD`、`POST`、`PUT`、`PATCH`、`DELETE`、`CONNECT`、`OPTIONS`、`TRACE`， `*` 表示任意类型 |
| rules.pattern            | string       | 请求的 URL Path 匹配规则，支持通配符，不允许有重叠匹配                                                                     |
| rules.flow               | string       | 规则匹配之后执行的处理流程，支持多个 flow 组合，依次顺序执行                                                               |
| permitted_client_ip_list | string array | 指定一组允许访客 IP 的白名单                                                                                               |
| denied_client_ip_list    | string array | 指定一组拒绝访客 IP 的黑名单                                                                                               |

## Pattern 语法

| 语法            | 说明                                          | 示例               |
| --------------- | --------------------------------------------- | ------------------ |
| {变量名}        | 带名称的变量                                  | `/{name}`          |
| {变量名:regexp} | 通过正则来限制变量的匹配规则                  | `/{name:[a-zA-Z]}` |
| {变量名:\*}     | 匹配之后的任意路径，只允许应用在 Pattern 末尾 | `/{any:*}`         |

更多示例：

```
Pattern: /user/{user}

 /user/gordon                     match
 /user/you                        match
 /user/gordon/profile             no match
 /user/                           no match

Pattern with suffix: /user/{user}_admin

 /user/gordon_admin               match
 /user/you_admin                  match
 /user/you                        no match
 /user/gordon/profile             no match
 /user/gordon_admin/profile       no match
 /user/                           no match


Pattern: /src/{filepath:*}

 /src/                     match
 /src/somefile.go          match
 /src/subdir/somefile.go   match
```

其他注意事项：

- Pattern 必须是 `/` 开头
- 任意匹配只能作为最后的一个规则

## IP 访问控制

如果希望对访问网关服务的来源 IP 进行访问控制，可以通过 `ip_access_control` 配置节点来进行管理。

```
router:
  - name: my_router
    default_flow: async_bulk
    ip_access_control:
      enabled: true
```

### 白名单

如果只希望某些特定指定 IP 的访客才能访问网关服务，可以在路由里面配置来实现访问准入，该请求会在链接建立的过程中直接拒绝。
如下例子，`133.37.55.22` 会被允许网关的服务访问，其余的 IP 都会拒绝。

```
router:
  - name: my_router
    default_flow: async_bulk
    ip_access_control:
      enabled: true
      client_ip:
        permitted:
         - 133.37.55.22
```

## 黑名单

如果希望拒绝某些特定指定 IP 的访客来访问网关服务，可以在路由里面配置来实现访问拒绝，该请求会在链接建立的过程中直接拒绝。
如下例子，`133.37.55.22` 就会被阻止网关的服务访问。

```
router:
  - name: my_router
    default_flow: async_bulk
    ip_access_control:
      enabled: true
      client_ip:
        denied:
         - 133.37.55.22
```
