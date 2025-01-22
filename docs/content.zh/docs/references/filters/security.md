---
title: "security"
---

# security

## 描述

security 过滤器用来对请求的 API 进行安全过滤，结合 Console 来进行统一的身份管理，包括鉴权和授权的集中化管控，同时支持与 LDAP 的身份集成。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: security_request
    filter:
      - security:
          elasticsearch: es-server
      - elasticsearch:
          elasticsearch: es-server
elastic:
  elasticsearch: es-server
  remote_configs: true
  health_check:
    enabled: false
  availability_check:
    enabled: false
  orm:
    enabled: true
    init_template: false
    init_schema: true
    index_prefix: ".infini_"

elasticsearch:
  - name: es-server
    enabled: true
    endpoints:
      - http://127.0.0.1:9200

security:
  enabled: true
  authc:
    realms:
      ldap:
#        test: #setup guide: https://github.com/infinilabs/testing/blob/main/setup/gateway/cases/elasticsearch/elasticsearch-with-ldap.yml
#          enabled: true
#          host: "localhost"
#          port: 3893
#          bind_dn: "cn=serviceuser,ou=svcaccts,dc=glauth,dc=com"
#          bind_password: "mysecret"
#          base_dn: "dc=glauth,dc=com"
#          user_filter: "(cn=%s)"
#          group_attribute: "ou"
#          bypass_api_key: true
#          cache_ttl: "10s"
#          role_mapping:
#            group:
#              superheros: [ "Administrator" ]
##            uid:
##              hackers: [ "Administrator" ]
        testing:
          enabled: true
          host: "ldap.forumsys.com"
          port: 389
          bind_dn: "cn=read-only-admin,dc=example,dc=com"
          bind_password: "password"
          base_dn: "dc=example,dc=com"
          user_filter: "(uid=%s)"
          cache_ttl: "10s"
          role_mapping:
            uid:
              tesla: [ "test-data" ]
```

## 参数说明

| 名称     | 类型   | 说明   |
| -------- | ------ | ------ |
| elasticsearch | string | Elasticsearch 集群实例名称 |

> 由于需要用到 Console 中配置的用户权限信息，elastic 模块下 elasticsearch 配置需要与 Console 配置的系统集群配置为同一个集群