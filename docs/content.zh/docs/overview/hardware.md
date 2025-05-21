---
weight: 40
title: "硬件规格"
---
# Easysearch 生产环境硬件配置推荐
在生产环境部署 Easysearch 时，高可用性 (HA) 是必须满足的核心要求。为实现完整的 HA 保障，您至少需要部署 3 个节点组成 Easysearch 集群。为获得最佳运维体验，建议配合使用 INFINI Console 和 Gateway，它们提供集群监控、告警和运维管理等完整功能，可大幅提升日常运维工作效率。
| Product | CPU | MEM | JVM | Disk | High Availability |
|:----|:---:|:---:|:---:|:---:| :---:|
| Easysearch | 16 | 64 | 31 | SSD | 3 |
| Console | 8 | 16 | - | >=50 GB, HDD or SSD | - |
| Gateway | 8 | 16 | - | >=50 GB, HDD or SSD | 2 |

针对存储配置，建议优先选用本地磁盘部署，Easysearch 容量规划应基于实际数据规模及业务场景进行合理配置。对于低负载集群或测试环境，可适当降低硬件资源配置标准，但需确保满足基础性能需求。生产环境推荐采用 SSD 存储，测试环境可选用性能较低的磁盘类型。