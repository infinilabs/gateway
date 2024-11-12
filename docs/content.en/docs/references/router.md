---
title: "Service Router"
weight: 30
---

# Service Router

INFINI Gateway judges the flow direction based on routers. A typical example of router configuration is as follows:

```
router:
  - name: my_router
    default_flow: default_flow
    tracing_flow: request_logging
    rules:
      - method:
          - PUT
          - POST
        pattern:
          - "/_bulk"
          - "/{index_name}/_bulk"
        flow:
          - bulk_process_flow
```

Router involves several important terms:

- Flow: Handling flow of a request. Flows can be defined in three places in a router.
- default_flow: Default handling flow, which is the main flow of service handling. Request forwarding, filtering, and caching are performed in this flow.
- tracing_flow: Flow used to track the request status. It is independent of the default_flow. This flow is used to log requests and collect statistics.
- rules: Requests are distributed to specific handling flows according to matching rules. Regular expressions can be used to match the methods and paths of requests.

## Parameter Description

| Name                     | Type         | Description                                                                                                                                                      |
| ------------------------ | ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| name                     | string       | Route name                                                                                                                                                       |
| default_flow             | string       | Name of the default request handling flow                                                                                                                        |
| tracing_flow             | string       | Name of the flow used to trace a request                                                                                                                         |
| rules                    | array        | List of routing rules to be applied in the array sequence                                                                                                        |
| rules.method             | string       | Method type of a request. The `GET`, `HEAD`, `POST`, `PUT`, `PATCH`, `DELETE`, `CONNECT`, `OPTIONS`, and `TRACE` types are supported and `*` indicates any type. |
| rules.pattern            | string       | URL path matching rule of a request. Patterns are supported and overlapping matches are not allowed.                                                             |
| rules.flow               | string       | Flow to be executed after rule matching. Multiple flows can be combined and they are executed sequentially.                                                      |
| permitted_client_ip_list | string array | Specified IP list will be allowed access to the gateway service, in order to permit specify user or application.                                                 |
| denied_client_ip_list    | string array | Specified IP list will not allowed access to the gateway service, in order to prevent specify user or application.                                               |

## Pattern Syntax

| Syntax                 | Description                                                                | Example            |
| ---------------------- | -------------------------------------------------------------------------- | ------------------ |
| {Variable name}        | Variable with a name                                                       | `/{name}`          |
| {Variable name:regexp} | Restricts the matching rule of the variable by using a regular expression. | `/{name:[a-zA-Z]}` |
| {Variable name:\*}     | Any path after matching. It can be applied only to the end of a pattern.   | `/{any:*}`         |

Examples:

```
Pattern: /user/{user}

 /user/gordon                     match
 /user/you                        match
 /user/gordon/profile             no match
 /user/                           no match

Pattern with suffix: /user/{user}_admin

 /user/gordon_admin               match
 /user/you_admin                  match
 /user/you                        no match
 /user/gordon/profile             no match
 /user/gordon_admin/profile       no match
 /user/                           no match


Pattern: /src/{filepath:*}

 /src/                     match
 /src/somefile.go          match
 /src/subdir/somefile.go   match
```

Notes:

- A pattern must begin with `/`.
- Any match is only used as the last rule.

## Permit IPs

If you only want some specific IP to access the gateway, you can configure it in the route section,
and the request will be directly rejected during the link establishment process. In the following example, `133.37.55.22` will be allowed to access the gateway, and the rest of the IP will be denied.

```
router:
  - name: my_router
    default_flow: async_bulk
    ip_access_control:
      enabled: true
      client_ip:
        permitted:
         - 133.37.55.22
```

## Block IPs

If you want to block specify know ip to access gateway, you can configure the ip list in the router section, the requests will be denied during the TCP connection establishment。
for below example，the ip `133.37.55.22` will be blocked:

```
router:
  - name: my_router
    default_flow: async_bulk
    ip_access_control:
      enabled: true
      client_ip:
        denied:
         - 133.37.55.22
```
