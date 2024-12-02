---
title: "ldap_auth"
---

# ldap_auth

## Description

The ldap_auth filter is used to set authentication based on the Lightweight Directory Access Protocol (LDAP).

## Configuration Example

A simple example is as follows:

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

The above configuration uses an online free [LDAP test server](https://www.forumsys.com/tutorials/integration-how-to/ldap/online-ldap-test-server/), the test user is `tesla`, and the password is `password`.

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

## Parameter Description

| Name            | Type     | Description                                                                                             |
| --------------- | -------- | ------------------------------------------------------------------------------------------------------- |
| host            | string   | Address of the LDAP server                                                                              |
| port            | int      | Port of the LDAP server. The default value is `389`.                                                    |
| tls             | bool     | Whether the LDAP server uses the Transport Layer Security (TLS) protocol. The default value is `false`. |
| bind_dn         | string   | Information about the user who performs the LDAP query                                                  |
| bind_password   | string   | Password for performing the LDAP query                                                                  |
| base_dn         | string   | Root domain for filtering LDAP users                                                                    |
| user_filter     | string   | Query condition for filtering LDAP users. The default value is `(uid=%s)`.                              |
| uid_attribute   | string   | Attribute of a user ID. The default value is `uid`.                                                     |
| group_attribute | string   | Attribute of a user group. The default value is `cn`.                                                   |
| attribute       | array    | List of attributes returned by the LDAP query                                                           |
| max_cache_items | int      | The max number of cached items                                                                          |
| cache_ttl       | duration | The expired TTL of cached items，default `300s`                                                         |
