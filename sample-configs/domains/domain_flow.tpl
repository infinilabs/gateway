flow:
  - name: flow_$[[CLUSTER_UUID]]
    filter:
      - elasticsearch:
          elasticsearch: es_$[[CLUSTER_UUID]]
          refresh:
            interval: 30s
            enabled: true

elasticsearch:
  - name: es_$[[CLUSTER_UUID]]
    enabled: true
    endpoints: $[[CLUSTER_ENDPOINTS]]
    basic_auth:
      username: $[[CLUSTER_USERNAME]]
      password: $[[CLUSTER_PASSWORD]]
    discovery:
      enabled: $[[CLUSTER_DISCOVERY_ENABLED]]
      refresh:
        enabled: true
        interval: 60s

