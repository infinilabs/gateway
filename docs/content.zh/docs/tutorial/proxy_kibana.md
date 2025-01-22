---
title: "为 Kibana 添加代理和基础安全"
weight: 100
---

# 为 Kibana 添加代理和基础安全

如果你的 Kibana 版本比较多或者比较旧，或者没有设置 TLS 和身份信息，那么任何人都有可能直接访问 Kibana，而使用极限网关可以快速的进行修复。

## 使用 HTTP 过滤器来转发请求

```
  - http:
      schema: "http" #https or http
      host: "192.168.3.188:5602"
```

## 添加身份验证

```
  - basic_auth:
      valid_users:
        medcl: passwd
```

## 在路由里面可以替换静态资源

```
  - method:
      - GET
    pattern:
      - "/plugins/kibanaReact/assets/illustration_integrations_lightmode.svg"
    flow:
      - replace_logo_flow
```

## 开启 TLS

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

## 完整配置如下

```
entry:
  - name: my_es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8000
      skip_occupied_port: true
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

## 效果如下

使用网关来访问 Kibana 就需要登陆了，如下：

{{% load-img "/img/kibana-login.jpg" "" %}}

登陆之后，可以看到，Kibana 里面的资源也被替换掉了，如下：

{{% load-img "/img/kibana-home.png" "" %}}

## 展望

通过极限网关，我们还可以挖掘更多玩法，比如可以替换 Kibana 里面的 Logo，
可以替换里面的 JS，可以替换里面的 CSS 样式，通过 JS 和 css 组合可以动态添加导航、页面、可视化等等。
