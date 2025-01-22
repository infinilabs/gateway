---
title: "在线过滤器"
weight: 80
bookCollapseSection: true
---

# 请求过滤器

## 什么是过滤器

过滤器是网关接收到请求之后，在流程里面定义的一系列处理单元，每个过滤器处理一件任务，可以灵活组合，过滤器是请求的在线处理。

## 过滤器列表

### 请求过滤

- [context_filter](./context_filter)
- [request_method_filter](./request_method_filter)
- [request_header_filter](./request_header_filter)
- [request_path_filter](./request_path_filter)
- [request_user_filter](./request_user_filter)
- [request_host_filter](./request_host_filter)
- [request_client_ip_filter](./request_client_ip_filter)
- [request_api_key_filter](./request_api_key_filter)
- [response_status_filter](./response_status_filter)
- [response_header_filter](./response_header_filter)

### 请求转发

- [context_switch](./context_switch)
- [ratio](./ratio)
- [clone](./clone)
- [switch](./switch)
- [flow](./flow)
- [redirect](./redirect)
- [hash_mod](./hash_mod)

### 请求干预

- [javascript](./javascript)
- [context_parse](./context_parse)
- [sample](./sample)
- [request_body_json_del](./request_body_json_del)
- [request_body_json_set](./request_body_json_set)
- [context_regex_replace](./context_regex_replace)
- [request_body_regex_replace](./request_body_regex_replace)
- [response_body_regex_replace](./response_body_regex_replace)
- [response_header_format](./response_header_format)
- [set_context](./set_context)
- [set_basic_auth](./set_basic_auth)
- [set_hostname](./set_hostname)
- [set_request_header](./set_request_header)
- [set_request_query_args](./set_request_query_args)
- [set_response_header](./set_response_header)
- [set_response](./set_response)

### 限速限流

- [context_limiter](./context_limiter)
- [bulk_request_throttle](./bulk_request_throttle)
- [request_path_limiter](./request_path_limiter)
- [request_host_limiter](./request_host_limiter)
- [request_user_limiter](./request_user_limiter)
- [request_api_key_limiter](./request_api_key_limiter)
- [request_client_ip_limiter](./request_client_ip_limiter)
- [retry_limiter](./retry_limiter)
- [sleep](./sleep)

### 日志监控

- [logging](./logging)

### Elasticsearch

- [date_range_precision_tuning](./date_range_precision_tuning)
- [elasticsearch_health_check](./elasticsearch_health_check)
- [bulk_response_process](./bulk_response_process)
- [bulk_request_mutate](./bulk_request_mutate)
- [auto_generate_doc_id](./auto_generate_doc_id)
- [rewrite_to_bulk](./rewrite_to_bulk)
- [request_reshuffle](./request_reshuffle)
- [bulk_reshuffle](./bulk_reshuffle)

### 身份认证

- [basic_auth](./basic_auth)
- [ldap_auth](./ldap_auth)

### Output

- [queue](./queue)
- [elasticsearch](./elasticsearch)
- [cache](./cache)
- [translog](./translog)
- [redis_pubsub](./redis_pubsub)
- [drop](./drop)
- [http](./http)

### 调试开发

- [echo](./echo)
- [dump](./dump)
- [record](./record)
