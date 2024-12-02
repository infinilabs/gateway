---
title: "Adding a TLS and Basic Security for Kibana"
weight: 100
---

# Adding a TLS and Basic Security for Kibana

If you have multiple Kibana versions or your Kibana version is out of date, or if you do not set TLS or identity, then anyone can directly access Kibana. You can use the INFINI Gateway to quickly fix this issue.

## Using the HTTP Filter to Forward Requests

```
  - http:
      schema: "http" #https or http
      host: "192.168.3.188:5602"
```

## Adding Authentication

```
  - basic_auth:
      valid_users:
        medcl: passwd
```

## Replacing Static Resources in the Router

```
  - method:
      - GET
    pattern:
      - "/plugins/kibanaReact/assets/illustration_integrations_lightmode.svg"
    flow:
      - replace_logo_flow
```

## Enabling TLS

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

## Complete Configuration

```
entry:
  - name: my_es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8000
    tls:
      enabled: true

flow:
  - name: logout_flow
    filter:
      - set_response:
          status: 401
          body: "Success logout!"
      - drop:
  - name: replace_logo_flow
    filter:
      - redirect:
          uri: https://elasticsearch.cn/uploads/event/20211120/458c74ca3169260dbb2308dd06ef930a.png
  - name: default_flow
    filter:
      - basic_auth:
          valid_users:
            medcl: passwd
      - http:
          schema: "http" #https or http
          host: "192.168.3.188:5602"
router:
  - name: my_router
    default_flow: default_flow
    rules:
      - method:
          - GET
          - POST
        pattern:
          - "/_logout"
        flow:
          - logout_flow
      - method:
          - GET
        pattern:
          - "/plugins/kibanaReact/assets/illustration_integrations_lightmode.svg"
        flow:
          - replace_logo_flow
```

## Effect

To access Kibana through INFINI Gateway, you need to log in as follows:

{{% load-img "/img/kibana-login.jpg" "" %}}

After login, you will find that resources in Kibana are also replaced. See the figure below.

{{% load-img "/img/kibana-home.png" "" %}}

## Prospect

We can explore other benefits of by using INFINI Gateway, for example, we can use the INFINI Gateway to replace the static assets, like logo, JS, and CSS style in Kibana, or use the combination of JS and CSS to dynamically add navigation and pages or advanced visualization.
