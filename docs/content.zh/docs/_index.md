---
title: INFINI Gateway
type: docs
bookCollapseSection: true
weight: 3
---

# 极限网关

## 介绍

**极限网关** (_INFINI Gateway_) 是一个面向 Elasticsearch 的高性能应用网关，它包含丰富的特性，使用起来也非常简单。极限网关工作的方式和普通的反向代理一样，我们一般是将网关部署在 Elasticsearch 集群前面，
将以往直接发送给 Elasticsearch 的请求都发送给网关，再由网关转发给请求到后端的 Elasticsearch 集群。因为网关位于在用户端和后端 Elasticsearch 之间，所以网关在中间可以做非常多的事情，
比如可以实现索引级别的限速限流、常见查询的缓存加速、查询请求的审计、查询结果的动态修改等等。

{{< button relref="./overview/" >}}了解更多{{< /button >}}

## 特性

> 极限网关是专为 Elasticsearch 而量身打造的应用层网关，地表最强，没有之一!

- 高可用，不停机索引，自动处理后端 Elasticsearch 的故障，不影响数据的正常摄取
- 写入加速，可自动合并独立的索引请求为批量请求，降低后端压力，提高索引效率
- 查询加速，可配置查询缓存，Kibana 分析仪表板的无缝智能加速，全面提升搜索体验
- 透明重试，自动处理后端 Elasticsearch 节点故障和对查询请求进行迁移重试
- 流量克隆，支持复制流量到多个不同的后端 Elasticsearch 集群，支持流量灰度迁移
- 一键重建，优化过的高速重建和增量数据的自动处理，支持新旧索引的透明无缝切换
- 安全传输，自动支持 TLS/HTTPS，可动态生成自签证书，也可指定自签可信证书
- 精准路由，多种算法的负载均衡模式，索引和查询可分别配置负载路由策略，动态灵活
- 限速限流，支持多种限速和限流测规则，可以实现索引级别的限速，保障后端集群的稳定性
- 并发控制，支持集群和节点级别的 TCP 并发连接数控制，保障后端集群和节点稳定性
- 无单点故障，内置基于虚拟 IP 的高可用解决方案，双机热备，故障自动迁移，避免单点故障
- 请求透视，内置日志和指标监控，可以对 Elasticsearch 请求做全面的数据分析

{{< button relref="./getting-started/install" >}}即刻开始{{< /button >}}

## 社区

[加入我们的 Discord Server](https://discord.gg/4tKTMkkvVX)

## 谁在用?

如果您正在使用极限网关，并且您觉得它还不错的话，请[告诉我们](https://discord.gg/4tKTMkkvVX)，所有的用户案例我们会集中放在[这里](./user-cases/)，感谢您的支持。
