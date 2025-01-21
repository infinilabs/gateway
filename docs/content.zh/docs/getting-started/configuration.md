---
weight: 30
title: 配置网关
---

# 配置

极限网关支持多种方式来修改配置。

## 命令行参数

极限网关提供了命令行参数如下：

```
✗ ./bin/gateway --help
Usage of ./bin/gateway:
  -config string
        the location of config file, default: gateway.yml (default "gateway.yml")
  -debug
        run in debug mode, gateway will quit with panic error
  -log string
        the log level,options:trace,debug,info,warn,error (default "info")
  -v    version
```

常用的说明如下：

- config，指定配置文件名，默认的配置文件名为当前执行命令所在目录的 `gateway.yml`，如果你的配置文件放置在其他地方，可以通过指定参数来进行选择。
- daemon，将网关切换到后台执行，一般还需要结合 `pidfile` 来保存进程号，方便后续的进程操作。

## 配置文件

极限网关的大部分配置都可以通过 `gateway.yml` 来进行配置，配置修改完成之后，需要重启网关程序才能生效。

### 定义入口

每一个网关都至少要对外暴露一个服务的入口，用来接收业务的操作请求，这个在极限网关里面叫做 `entry`，通过下面的参数即可定义：

```
entry:
  - name: es_gateway
    enabled: true
    router: default
    network:
      binding: 0.0.0.0:8000
```

这里定义了一个名为 `es_gateway` 的服务入口，监听的地址是 `0.0.0.0:8000`，使用了一个名为 `default` 的路由来处理请求。

### 定义路由

极限网关通过路由来判断流量的去向，一个典型的路由配置示例如下：

```
router:
  - name: default
    default_flow: cache_first
```

这里定义了一个名为 `default` 的路由，也就是业务处理的主流程，请求转发、过滤、缓存等操作都在这里面进行。

### 定义流程

一个请求流程定义了一系列请求处理的工作单元，是一个典型的管道式工作方式，一个典型的配置示例如下：

```
flow:
  - name: cache_first
    filter:
      - get_cache:
      - elasticsearch:
          elasticsearch: prod
      - set_cache:
```

上面的配置定义了一个名为 `cache_first` 的处理流，使用了三个不同的 filter，分别是 `get_cache`、`elasticsearch` 和 `set_cache`，这些 filter 会依据配置的先后顺序依次执行，注意每个 filter 名称后面要带上一个 `:`。
各个 filter 的处理结果分别如下：

- get_cache，这个 filter 主要用来从缓存里面拿数据，如果之前发生过相同的请求，并且缓存还存在且有效的情况下，这个 filter 可以直接拿到缓存然后立即返回，不用继续往下处理；
- elasticsearch，这个 filter 主要用来将请求转发给后端的 Elasticsearch 集群，并且将 Elasticsearch 返回的响应内容继续往下传递；
- set_cache，这个 filter 会将执行结果缓存到本地内存，有一些参数限制，比如状态码，请求大小等，并设置一定的过期时间，以方便下次重复请求可以直接使用缓存，一般要和 `get_cache` 组合使用。

### 定义资源

这里的资源主要是指 Elasticsearch 后端服务器资源，极限网关支持多个 Elasticsearch 集群，可以实现将请求转发到多个不同集群，也可以支持请求的蓝绿发布、灰度切换等，定义一个 Elasticsearch 后端资源的方式示例如下：

```
elasticsearch:
- name: prod
  enabled: true
  endpoint: http://192.168.3.201:9200
  discovery:
    enabled: true
    refresh:
      enabled: true
  basic_auth:
    username: elastic
    password: pass
```

通过参数 `endpoint` 来设置 Elasticsearch 的访问地址，如果 Elasticsearch 开启了身份认证，可以通过 `basic_auth` 来指定用户名和密码信息，该用户需要有能够获取集群状态信息的权限。
通过参数 `discover` 可以开启自动的后端节点的自动发现，用于自动检测后端节点的情况，能够自动识别新增和离线的节点。

通过这些基本的配置，我们就可以正常的代理 Elasticsearch 的请求了，关于每个组件更详细完整的参数，请参考[功能手册](../references/filters/)。
