---
weight: 36
title: K8s Deployment
asciinema: true
draft: true
---

# Deploying the K8s Environment

INFINI Gateway can be also deployed in the K8s environment.

## Creating a Gateway Service

Edit a deployment configuration file `vim my-deployment.yml` as follows:

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

Run the following command to create a service for INFINI Gateway:

```
kubectl create -f my-deployment.yml
```

If the service is in good condition, three gateway service instances will be created. Run the following commands to check their running status:

```
kubectl get deployment | grep infini-gateway
kubectl get replicaset | grep infini-gateway
kubectl get pod | grep infini-gateway
```

## Creating an External Service

Use the NodePort mode to externally keep the gateway service and add the configuration file `vim my-service.yml` with the following content:

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

Run the following command to create a service:

```
kubectl create -f my-service.yml
```

Run the following commands to display the service:

```
kubectl get service | grep infini-gateway
kubectl describe service infini-gateway
```
