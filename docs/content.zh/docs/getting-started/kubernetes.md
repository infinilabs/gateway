---
weight: 36
title: k8s 部署
asciinema: true
draft: true
---

# k8s 环境部署

极限网关也支持部署在 k8s 环境上。

## 创建网关服务

编辑一个部署配置 `vim my-deployment.yml`，内容如下：

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: infini-gateway
spec:
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: infini-gateway
  replicas: 3
  template:
    metadata:
      labels:
        app: infini-gateway
    spec:
      containers:
      - name: infini-gateway
        image: infinilabs/gateway
        ports:
        - containerPort: 8000
```

执行如下命令来创建该极限网关的服务：

```
kubectl create -f my-deployment.yml
```

如果一切正常，应该会创建 3 个网关的服务实例，可以通过如下命令来查看运行状态：

```
kubectl get deployment | grep infini-gateway
kubectl get replicaset | grep infini-gateway
kubectl get pod | grep infini-gateway
```

## 创建对外的服务

使用 NodePort 模式对外保留网关的服务，新增配置文件 `vim my-service.yml`，内容如下：

```
apiVersion: v1
kind: Service
metadata:
  name: infini-gateway
  namespace: default
  labels:
    app: infini-gateway
spec:
  externalTrafficPolicy: Local
  ports:
  - name: http
    port: 8000
    protocol: TCP
    targetPort: 8000
  selector:
    app: infini-gateway
  type: NodePort
```

执行如下命令创建服务：

```
kubectl create -f my-service.yml
```

使用如下命令来查看服务:

```
kubectl get service | grep infini-gateway
kubectl describe service infini-gateway
```
