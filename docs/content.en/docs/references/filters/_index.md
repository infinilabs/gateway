---
title: "Online Filter"
weight: 80
bookCollapseSection: true
---

# Request Filter

## What Is a Filter

A filter is a series of processing units defined in a flow for requests received by the gateway. Each filter processes one task and the filters can be flexibly combined. Filters process requests online.

## Filter List

### Request Filtering

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

### Request Forwarding

- [context_switch](./context_switch)
- [ratio](./ratio)
- [clone](./clone)
- [switch](./switch)
- [flow](./flow)
- [redirect](./redirect)
- [hash_mod](./hash_mod)

### Request Mutation

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

### Traffic Control and Throttling

- [context_limiter](./context_limiter)
- [bulk_request_throttle](./bulk_request_throttle)
- [request_path_limiter](./request_path_limiter)
- [request_host_limiter](./request_host_limiter)
- [request_user_limiter](./request_user_limiter)
- [request_api_key_limiter](./request_api_key_limiter)
- [request_client_ip_limiter](./request_client_ip_limiter)
- [retry_limiter](./retry_limiter)
- [sleep](./sleep)

### Log Monitoring

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


### Authentication

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

### Debugging and Development

- [echo](./echo)
- [dump](./dump)
- [record](./record)
