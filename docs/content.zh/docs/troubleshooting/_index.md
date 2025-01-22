---
title: "常见问题"
weight: 50
---

# 常见问题及故障处理

这里主要收集极限网关使用过程中遇到的常见问题及处理办法，欢迎反馈提交到 [这里](https://github.com/infinilabs/gateway/issues/new/choose) 。

## 常见问题

### 服务启动不了

问题描述: 安装系统服务但是启动失败
问题解答: 服务失败的原因有很多，为了帮助我们快速定位问题，请尝试执行 `journalctl -xeu gateway`、`dmesg`、`tail -n 1000 /var/log/syslog`
命令来获取服务启动的相关失败日志信息，同时提供配置文件、程序版本信息，
并在 [这里](https://github.com/infinilabs/gateway/issues/new/choose) 反馈给我们。

### 配置里面 Elasticsearch 的身份信息

问题描述：我看到在配置 Elasticsearch 的时候，需要指定用户信息，有什么用，可以不配么？
问题解答：极限网关是透明网关，使用网关之前是怎么传的参数，在替换为网关之后，还是照样传，比如身份信息还是需要传递。
网关配置里面的身份信息主要用于获取集群的内部运行状态和元数据，一些异步的操作或需要由网关来进行集群的操作也需要使用到该身份信息，比如记录指标和日志。

### 写入速度没有提升

问题描述：为什么我用了极限网关的 `bulk_reshuffle`，写入速度没有提升呢？

问题解答：如果你的集群节点总数太少，比如低于 `10` 个数据节点或者索引吞吐低于 `15w/s`，你可能没有必要使用这个功能或者关注点不应该在写入性能上面，
因为集群规模太小，Elasticsearch 因为转发性能和请求分发造成的影响不是特别明显，走不走网关理论上性能不会差距很大。
当然使用 `bulk_reshuffle` 还有其他好处，比如数据先落地网关队列可以解耦后端 Elasticsearch 故障的影响。

### Elasticsearch 401 错误

问题描述：访问网关提升身份验证失败
问题解答：极限网关是透明网关，在网关配置里面的身份信息仅用于网关和 Elasticsearch 的通信，客户端通过网关来访问 Elasticsearch 任然需要传递适当的身份信息。

## 常见故障

### 端口重用不支持的问题

错误提示：The OS doesn't support SO_REUSEPORT: cannot enable SO_REUSEPORT: protocol not available

问题描述：极限网关默认开启端口重用，用于多进程共享端口，在旧版本的 Linux 内核中需要打补丁才能使用。

解决方案：可以通过修改监听网络的配置，将 `reuse_port` 改成 `false`，关闭端口重用：

```
**.
   network:
     binding: 0.0.0.0:xx
     reuse_port: false
```

### Elasticsearch 用户权限不够

错误提示：[03-10 14:57:43] [ERR] [app.go:325] shutdown: json: cannot unmarshal object into Go value of type []adapter.CatIndexResponse

问题描述：极限网关 Elasticsearch 配置开启 discovery 的情况下，如果用户权限给的不够，会提示这个错误，因为需要访问相关的 Elasticsearch API 来获取集群的信息。

解决方案：给相关的 Elasticsearch 用户赋予所有索引的 `monitor` 和 `view_index_metadata` 权限即可。
