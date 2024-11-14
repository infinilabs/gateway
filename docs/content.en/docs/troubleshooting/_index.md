---
title: "FAQs"
weight: 50
---

# FAQs and Troubleshooting

FAQs about INFINI Gateway and handling methods are provided here. You are welcome to submit your problems [here](https://github.com/infinilabs/gateway/issues/new/choose).

## FAQs

### The Auth section in Elasticsearch configuration

Q：I see that when configuring Elasticsearch, I need to specify user information, what is use for?

A：INFINI Gateway is a transparent gateway, the parameters that are passed before the gateway placed,
just keep them same, such as identity information or any other parameters that need to be passed.
The auth information in the gateway mainly used to obtain the internal running status or metadata info of the cluster,
some asynchronous operations that require the gateway to carry out also need to use the auth information,
such as persist metrics or logs to that elasticsearch cluster.

### The Write Speed Is Not Improved

Q: Why is the write speed not improved after I use `bulk_reshuffle` of the INFINI gateway?

A: If your cluster has a small number of nodes, for example, if it contains less than `10` data nodes or if the index throughput is lower than `150k event/s`, you may not need to use this feature or you do not need to focus on the write performance because the cluster size is too small and forwarding and request distribution have a minimal impact on Elasticsearch.
Therefore, the performance does not differ greatly regardless of whether the gateway is used.
But there are other benefits of using `bulk_reshuffle`, for example, the impact of faults occurring on the back-end Elasticsearch can be decoupled if data is sent to the gateway first.

### Elasticsearch 401 Error

Q：I see some error when switch to INFINI Gateway: `missing authentication credentials for REST request [/]`

A：The INFINI gateway is a transparent gateway. The auth information configured in the gateway only used for the communication between the gateway and Elasticsearch. Clients still need to pass appropriate auth information to access Elasticsearch resources.

## Common Faults

### Port Reuse Is Not Supported

Error prompt: The OS doesn't support SO_REUSEPORT: cannot enable SO_REUSEPORT: protocol not available

Fault description: Port reuse is enabled on INFINI Gateway by default. It is used for multi-process port sharing. Patches need to be installed in the Linux kernel of the old version so that the port reuse becomes available.

Solution: Modify the network monitoring configuration by changing `reuse_port` to `false` to disable port reuse.

```
**.
   network:
     binding: 0.0.0.0:xx
     reuse_port: false
```

### An Elasticsearch User Does Not Have Sufficient Permissions

Error prompt: [03-10 14:57:43] [ERR] [app.go:325] shutdown: json: cannot unmarshal object into Go value of type []adapter.CatIndexResponse

Fault description: After discovery is enabled for Elasticsearch on INFINI Gateway, this error is generated if the user permission is insufficient. The cause is that relevant Elasticsearch APIs need to be accessed to acquire cluster information.

Solution: Grant the `monitor` and `view_index_metadata` permissions of all indexes to relevant Elasticsearch users.
