---
title: "request_host_filter"
---

# request_host_filter

## Description

The request_host_filter is used to filter requests based on a specified domain name or host name. It is suitable for scenarios in which there is only one IP address but access control is required for multiple domain names.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test
    filter:
      - request_host_filter:
          include:
            - domain-test2.com:8000
```

The above example shows that only requests that are used to access the domain name `domain-test2.com:8000` are allowed to pass through.

## Example

```
✗ curl -k -u user:passwd http://domain-test4.com:8000/   -v

*   Trying 192.168.3.67...
* TCP_NODELAY set
* Connected to domain-test4.com (192.168.3.67) port 8000 (#0)
* Server auth using Basic with user 'medcl'
> GET / HTTP/1.1
> Host: domain-test4.com:8000
> Authorization: Basic 123=
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 403 Forbidden
< Server: INFINI
< Date: Fri, 15 Jan 2021 13:53:01 GMT
< Content-Length: 0
< FILTERED: true
<
* Connection #0 to host domain-test4.com left intact
* Closing connection 0

✗ curl -k -u user:passwd http://domain-test2.com:8000/   -v

*   Trying 192.168.3.67...
* TCP_NODELAY set
* Connected to domain-test2.com (192.168.3.67) port 8000 (#0)
* Server auth using Basic with user 'medcl'
> GET / HTTP/1.1
> Host: domain-test2.com:8000
> Authorization: Basic 123=
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 200 OK
< Server: INFINI
< Date: Fri, 15 Jan 2021 13:52:53 GMT
< Content-Type: application/json; charset=UTF-8
< Content-Length: 480
< UPSTREAM: 192.168.3.203:9200
< CACHE-HASH: a2902f950b4ade804b21a062257387ef
<
{
  "name" : "node3",
  "cluster_name" : "pi",
  "cluster_uuid" : "Z_HcN_6ESKWicV-eLsyU4g",
  "version" : {
    "number" : "6.4.2",
    "build_flavor" : "default",
    "build_type" : "tar",
    "build_hash" : "04711c2",
    "build_date" : "2018-09-26T13:34:09.098244Z",
    "build_snapshot" : false,
    "lucene_version" : "7.4.0",
    "minimum_wire_compatibility_version" : "5.6.0",
    "minimum_index_compatibility_version" : "5.0.0"
  },
  "tagline" : "You Know, for Search"
}
* Connection #0 to host domain-test2.com left intact
* Closing connection 0
```

## Parameter Description

| Name    | Type   | Description                                                                                                                              |
| ------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| exclude | array  | List of hosts, from which requests are refused to pass through                                                                           |
| include | array  | List of hosts, from which requests are allowed to pass through                                                                           |
| action  | string | Processing action after filtering conditions are met. The value can be set to `deny` or `redirect_flow` and the default value is `deny`. |
| status  | int    | Status code returned after the user-defined mode is matched                                                                              |
| message | string | Message text returned in user-defined `deny` mode                                                                                        |
| flow    | string | ID of the flow executed in user-defined `redirect_flow` mode                                                                             |

{{< hint info >}}
Note: If the `include` condition is met, requests are allowed to pass through only when at least one response code in `include` is met.
If only the `exclude` condition is met, any request that does not meet `exclude` is allowed to pass through.
{{< /hint >}}
