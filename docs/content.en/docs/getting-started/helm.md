---
weight: 40
title: Kubernetes Deployment
asciinema: true
---

# Helm Charts

INFINI Gateway supports deployment on K8s by using helm chart .

## The Chart Repository

Chart repository: [https://helm.infinilabs.com](https://helm.infinilabs.com/).

Use the follow command add the repository:

```bash
helm repo add infinilabs https://helm.infinilabs.com
```

## Prerequisites

- K8S StorageClass

The default StorageClass of the Chart package is local-path, you can install it through [here](https://github.com/rancher/local-path-provisioner).

If you want use other StorageClass(installed), you can create a YAML file (eg. vaules.yaml) file that it contains the follow contents:
```yaml
storageClassName: \<storageClassName\>
```
and use it through `-f`.

- Storage Cluster

The default Storage Cluster of the Chart package is Easysearch, you can install it through [here](https://www.infinilabs.com/docs/latest/easysearch/getting-started/install/helm/).
```
Note: The username and password of easysearch in the Chart package is default, if you change it, you can adjust this by modifying the cluster connection below。
```

Gateway also support other cluster (eg. Elasticsearch、Opensearch)，you can create a YAML file (eg. vaules.yaml) file that it contains the follow contents:
```yaml
env:
  # connection address of the logging cluster
  loggingEsEndpoint: ******
  # username of the logging cluster
  loggingEsUser: ******
  # password of the logging cluster's user
  loggingEsPass: ******
  # connection address of the production cluster
  prodEsEndpoint: ******
  # username of the production cluster
  prodEsUser: ******
  # password of the production cluster's user
  prodEsPass: ******
```
and use it through `-f`.

## Install

```bash
helm install gateway infinilabs/gateway -n <namespace>
```

## Uninstall

```bash
helm uninstall gateway -n <namespace>
kubectl delete pvc gateway-data-gateway-0 -n <namespace>
```