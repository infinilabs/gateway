  - ssl
  - jwt
  - acl
  - cors
  - oauth2
  - tcp-log
  - udp-log
  - file-log
  - http-log
  - key-auth
  - hmac-auth
  - basic-auth
  - ip-restriction
  - mashape-analytics
  - request-transformer
  - response-transformer
  - request-size-limiting
  - rate-limiting
  - response-ratelimiting
  - search_timeout
  - max_search_size
  - max_term_size
  - timeout(join type)
  - max_retry(join type)
  - slow_logging(join type)

  

# Rules
| METHOD | PATH | FLOW                                                         |
| ------ | ---- | ------------------------------------------------------------ |
| GET    | /    | name=cache_first flow =[ get_cache >> forward >> set_cache ] |
| GET    | /_cat/*item     | name=forward flow=[forward]                                                             |
| POST \|\| PUT | /:index/\_doc/*id | name=forward flow=[forward]                                  |
| POST \|\| PUT | /:index/\_bulk \|\| /\_bulk     |  name=async_indexing_via_translog flow=[ save_translog ]                                                            |
| POST \|\| GET	|  /:index/\_search				|  name=cache_first flow=[ get_cache >> forward >> set_cache ]  |
| POST \|\| PUT		|  /:index/\_bulk \|\| /\_bulk 	|  name=async_dual_writes flow=[ save_translog{name=dual_writes_id1, retention=7days, max_size=10gb} ]  |
| POST \|\| PUT		| /:index/\_bulk \|\| /\_bulk 	| name=sync_dual_writes flow=[ mirror_forward ]  |
| GET				| /audit/*operations			| name=secured_audit_access flow=[ basic_auth >> flow{name=cache_first} ]  |


# Services
- Scheduler task check
- Bulk indexing
- Async traffic mirror
- Indexing requests merger
- Request logging
- Router DAG rules offline build
- Elasticsearch node health check
- Floating IP HA check
- License check
- System capacity check
- Circuit breaker check
- Metrics collector
