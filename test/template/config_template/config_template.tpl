entry:
  - name: my_es_entry_$[[name]]
    enabled: true
    router: my_router_$[[name]]
    max_concurrency: 10000
    network:
      binding: $[[binding_host]]
    tls:
      enabled: $[[tls_on_entry]]

flow:
  - name: es-flow_$[[name]]
    filter:
      - elasticsearch:
          elasticsearch: es-server_$[[name]]

router:
  - name: my_router_$[[name]]
    default_flow: es-flow_$[[name]]

elasticsearch:
  - name: es-server_$[[name]]
    enabled: true
    endpoints:
     - $[[elasticsearch_endpoint]]
