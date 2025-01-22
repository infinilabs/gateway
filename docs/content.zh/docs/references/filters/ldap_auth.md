---
title: "ldap_auth"
---

# ldap_auth

## 描述

ldap_auth 过滤器用来设置基于 LDAP 的身份认证。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: ldap_auth
    filter:
      - ldap_auth:
          host: "ldap.forumsys.com"
          port: 389
          bind_dn: "cn=read-only-admin,dc=example,dc=com"
          bind_password: "password"
          base_dn: "dc=example,dc=com"
          user_filter: "(uid=%s)"
```

上面的配置使用的是在线的免费 [LDAP 测试服务器](https://www.forumsys.com/tutorials/integration-how-to/ldap/online-ldap-test-server/)，测试用户 `tesla`，密码 `password`。

```
➜  curl  http://127.0.0.1:8000/ -u tesla:password
{
  "name" : "192.168.3.7",
  "cluster_name" : "elasticsearch",
  "cluster_uuid" : "ZGTwWtBfSLWRpsS1VKQDiQ",
  "version" : {
    "number" : "7.8.0",
    "build_flavor" : "default",
    "build_type" : "tar",
    "build_hash" : "757314695644ea9a1dc2fecd26d1a43856725e65",
    "build_date" : "2020-06-14T19:35:50.234439Z",
    "build_snapshot" : false,
    "lucene_version" : "8.5.1",
    "minimum_wire_compatibility_version" : "6.8.0",
    "minimum_index_compatibility_version" : "6.0.0-beta1"
  },
  "tagline" : "You Know, for Search"
}
➜  curl  http://127.0.0.1:8000/ -u tesla:password1
Unauthorized%
```

## 参数说明

| 名称            | 类型     | 说明                                             |
| --------------- | -------- | ------------------------------------------------ |
| host            | string   | LDAP 服务器地址                                  |
| port            | int      | LDAP 服务器端口，默认 `389`                      |
| tls             | bool     | LDAP 服务器是否为 TLS 安全传输协议，默认 `false` |
| bind_dn         | string   | 执行 LDAP 查询的用户信息                         |
| bind_password   | string   | 执行 LDAP 查询的密码信息                         |
| base_dn         | string   | 过滤 LDAP 用户的根域                             |
| user_filter     | string   | 过滤 LDAP 用户的查询条件，默认 `(uid=%s)`        |
| uid_attribute   | string   | 用于用户 ID 的属性，默认 `uid`                   |
| group_attribute | string   | 用于用户组的属性，默认 `cn`                      |
| attribute       | array    | 指定 LDAP 查询返回的属性列表                     |
| max_cache_items | int      | 最大的缓存格式，默认不限制                       |
| cache_ttl       | duration | 缓存过期时间格式，默认 `300s`                    |
