---
title: "Enable HTTPS/TLS + Basic Auth for Elasticsearch easily"
weight: 100
---

# Enable HTTPS/TLS + Basic Auth for Elasticsearch easily

If you have multiple Elasticsearch versions or your version is out of date, or if you do not set TLS or identity, then anyone can directly access Elasticsearch. You can use INFINI Gateway to quickly fix this issue.

## Define an Elasticsearch resource

Let's define the Elasticsearch resources, config as bellow：

```
elasticsearch:
- name: prod
  enabled: true
  endpoint: http://192.168.3.201:9200
```

The `prod` refer to `http://192.168.3.201:9200`

And then, we will need to use a filter to forward requests to that Elasticsearch，which name is `prod`：

```
  - elasticsearch:
      elasticsearch: prod
```

For more options of this elasticsearch filter, please refer to documentation：[elasticsearch filter](../references/filters/elasticsearch/)

## Add basic_auth filter

In order to perform access control of elasticsearch, we are using a basic_auth filter for example:

```
  - basic_auth:
      valid_users:
        medcl: passwd
```

The only valid user defined in above configuration.

## Enable TLS

Enable auth, but do not enable the TLS, it is useless, because HTTP is a clear text transmission protocol,
which can easily leak the passwords, enable the TLS is quite simple, jut define a entry as below:

```
  - name: my_es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8000
    tls:
      enabled: true
```

You can try visit `https://localhost:8000` to access the `prod` Elasticsearch cluster now。

Note that the listening address here is '0.0.0.0', which means that the IP on all the network cards on the machine are listening.
For security reasons, you may need to change to listen only on local addresses or specified NIC IP addresses.

## Compatible with HTTP access

If there are legacy systems that cannot switch to HTTPS, we can leverage gateway to provide plain HTTP access too:

```
  - name: my_unsecure_es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8001
    tls:
      enabled: false
```

By visit `http://localhost:8001` you can access the `prod` cluster too。

## Full configuration

```
elasticsearch:
- name: prod
  enabled: true
  endpoint: http://192.168.3.201:9200

entry:
  - name: my_es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8000
    tls:
      enabled: true
  - name: my_unsecure_es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8001
    tls:
      enabled: false

flow:
  - name: default_flow
    filter:
      - basic_auth:
          valid_users:
            medcl: passwd
      - elasticsearch:
          elasticsearch: prod
router:
  - name: my_router
    default_flow: default_flow
```

## Showcase

You will a valid user to access Elasticsearch now：

{{% load-img "/img/elasticsearch-login.jpg" "" %}}
