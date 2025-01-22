---
weight: 36
title: Helm 部署
asciinema: true
---

# Helm 部署

INFINI Gateway 支持 Helm 方式部署。

## 添加仓库

Chart 仓库地址在这里 [https://helm.infinilabs.com](https://helm.infinilabs.com/)。

使用下面的命令添加仓库：

```bash
helm repo add infinilabs https://helm.infinilabs.com
```

## 前提

- K8S StorageClass

Chart 包中默认配置的 StorageClass 是 local-path，可参考[这里](https://github.com/rancher/local-path-provisioner)安装。

如果想使用其他已安装的 StorageClass, 可以创建一个 YAML 文件（例如：values.yaml），添加如下配置：
```yaml
storageClassName: <storageClassName>
```
创建的时候使用 `-f` 参数指定，替换默认值。

- 存储集群

Chart 包中配置的默认存储集群是 Easysearch，可参考[这里](https://www.infinilabs.com/docs/latest/easysearch/getting-started/install/helm/)安装。
```
注：Chart 包中配置的用户名和密码也是默认的，如有变动，可参照下面修改集群连接地址方法进行调整。
```

Gateway 也支持其他集群（如 Elasticsearch、Opensearch）连接，需手动创建一个 YAML 文件（例如：values.yaml），添加如下配置：
```yaml
env:
  # 请求记录存储集群
  loggingEsEndpoint: ******
  # 请求记录存储集群用户
  loggingEsUser: ******
  # 请求记录存储集群用户密码
  loggingEsPass: ******
  # 业务存储集群
  prodEsEndpoint: ******
  # 业务存储集群用户
  prodEsUser: ******
  # 业务存储集群用户密码
  prodEsPass: ******
```
创建的时候使用 `-f` 参数指定，替换默认值。

## 安装

```bash
helm install gateway infinilabs/gateway -n <namespace>
```

## 卸载

```bash
helm uninstall gateway -n <namespace>
kubectl delete pvc gateway-data-gateway-0 -n <namespace>
```