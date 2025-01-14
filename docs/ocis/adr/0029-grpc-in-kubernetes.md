---
title: "29. gRPC in Kubernetes"
date: 2024-06-27T14:05:00+01:00
weight: 29
geekdocRepo: https://github.com/owncloud/ocis
geekdocEditPath: edit/master/docs/ocis/adr
geekdocFilePath: 0029-grpc-in-kubernetes.md
---

* Status: draft
* Deciders: [@butonic](https://github.com/butonic)
* Date: 2024-06-27

## Context and Problem Statement

[Scaling oCIS in kubernetes causes requests to fail
#8589](https://github.com/owncloud/ocis/issues/8589), sometimes until affected services have manually been restarted. This can be observed by two symptoms:
1. when a new pod is added it does not receive traffic
2. when an pod is shut down existing clients still try to send requests to it

To leverage the kubernetes pod state, we first used the go micro kubernetes registry implementation. When a pod fails the health or readyness probes, kubernetes will no longer
- send traffic to the pod via the kube-proxy, which handles the ClusterIP for a service, 
- list the pod in DNS responses when the ClusterIP is disabled by setting it to `none`
When using the ClusterIP HTTP/1.1 requests will be routed to a working pod. 

This nice setup starts to fail with long lived connections. The kube-proxy is connection based, causing requests with Keep-Alive to stick to the same pod for more than one request. Worse, HTTP/2 and in turn gRPC are multiplexing the connection. They will not pick up any changes to pods, explaining the symptomps:
1. new pods will not be used because clients will reuse the existing gRPC connection
2. gRPC clients will still try to send traffic to killed pods because they have not picked up that the pod was killed. Or the pod was killed a millisecond after the lookup was made.

An addition to this problem are the health and readyness implementations of oCIS services not always reflecting the correct state of the service. One example is the storage-users service that returns ready `true` while runing a migration on startup.

Furthermore, the go micro kubernetes registry put too much load on the etcd service registry / kubernetes API. Maybe, because every pod keeps a connection open and is sent events ... causing a lot of traffic when multiple oCIS deployments are running in the same kubernetes cluster. Admittedly, that explanation needs to be verified, but keep this problem in mind as the possible solutions will have to deal with the same root cause.

Other reasons for the kubernetes API not being available are cluster upgrades of kubernetes itself, sometimes leading to the kubernetes API being unavailable for minutes. As a result, relying on the kubernetes API to do service lookup may disrupt the service operation if the service registry implementation cannot handle this kind of downtime.

To take the load off the kubernetes API we now roll our own nats-js-kv based service registry. It works, but we now ignore the readyness and health probes that are made by kubernetes. The go micro client would however address the problems:
1. the selector.Select() call will fetch a client based on the micro registry implementation. All registry implementations subscribe to changes, so new pods will be picked up. The kubernetes registry implementation even takes into account the pod ready probes, so this should be the right solution. As mentioned above, this seems to cause too much load on the kubernetes API. The nats-js-kv registry is aware of pods even if they are not ready yet, leading to race conditions and failed requests when a pod is added. This is mitigated somewhat by the next go micro client feature
2. by default, requests are retried five times when they fail. This mainly addresses failed requests when a pod is killed because the client will make a selector.Select() call to find a working connection. It also helps with pods that have been registered but are not ready, yet.

Unfortunately, reva does not use go micro grpc clients. We implemented our own selector mechanism for the reva pool and always select the next client. This gives us an ip that is used by the upstream grpc-go client to dial the connection. No retries are configured, yet.

What makes this worse is that we cannot use the native retry mechanism of grpc-go, because we already looked up an ip and the pod might already have been killed and we would just retry sending requests to the same ip.

To top it all off, go-micro V5 has changed the license to BSL and back to Apache 2, which raises concerns on how long we can safely rely on it.

We need to decide how we want to load balance and retry grpc connections in kubernetes.

## Decision Drivers

* oCIS should scale in Kubernetes without losing requests
* The code should be maintainable
* Connections should work on localhost as well as in kubernetes

## Considered Options
* go-micro clients
* Proxy load balancing
* Thick client-side load balancing
* Lookaside Load Balancing

## Decision Outcome
### Positive Consequences:
### Negative consequences:

## Pros and Cons of the Options
### go-micro clients
* good, because we can use the go micro client retry mechanism
* good, because we keep using interfaces that allow us to change the implementatien to test nats-js-kv vs kubernetes or whatever
* bad, pod readiness in kubernetes is basically ignored
* bad, because we need to change every line of code in reva that makes a grpc call

### [Proxy load balancing](https://grpc.io/blog/grpc-load-balancing/#proxy-load-balancer-options)
We could use a L7 proxy like envoy or linkerd to do the gRPC load balancing.  kubernetes would have to return all pod ips in dns responses by setting `clusterIP: None` to use headless services. And clients would have to use the envoy proxy address
* good, because clients are simple
* bad, because proxy adds an extra hop
* bad, because it consumes extra resources (10mb per pod, +1ms request latency)

### [Thick client-side load balancing](https://grpc.io/blog/grpc-load-balancing/#thick-client)
For this we would have to replace the go micro service registry and rely on dns as a service registry. service names would have to be configured with a schema, eg. `dns:///localhost:9142`, `dns:///gateway.ocis.svc.cluster.local:9142` or `unix:/var/run/gateway.socket` and kubernetes would have to return all pod ips in dns responses by setting `clusterIP: None` to use headless services.
* good, because we can use the grpc-go native retry mechanism
* good, because pod readyness is respected
* good, because we get rid of the complexity of a service registry - which means revisiting [ADR0006 Service Discovery](https://owncloud.dev/ocis/adr/0006-service-discovery/)
* bad, because we would lose the service registry - which migh also be good, see above
* bad, because every client will hammer the dns, maybe causing a similar load problem as with the go micro kubernetes registry implementation - needs performance testing

There are two pull requests that allow configuring ocis and reva to test this option:
* [respect grpc service transport cs3org/reva#4744](https://github.com/cs3org/reva/pull/4744)
* [set the configured protocol transport for service metadata owncloud/ocis#9490](https://github.com/owncloud/ocis/pull/9490)

### [Lookaside Load Balancing](https://grpc.io/blog/grpc-load-balancing/#lookaside-load-balancing)
* good, because the blog recommends it for very high performance requirements (low latency, high traffic)
* bad, because it seems to add even more complexity
* bad, because we need to spend time to research it

## Links
* Learnk8s - Jun 2024 - [Load balancing and scaling long-lived connections in Kubernetes](https://learnk8s.io/kubernetes-long-lived-connections) - explains why the kube-proxy does not fit long lived connections ... as in grpc - recommends client-side load balancing
* Medixm - Mar 2024 - [Don’t Load Balance GRPC or HTTP2 Using Kubernetes Service](https://medium.com/@lapwingcloud/dont-load-balance-grpc-or-http2-using-kubernetes-service-ae71be026d7f) - compares headless service with client side load balancing, ingress nginx and service mesh
* Dev.to - Jun 2023 - [Basic Guide to Kubernetes Service Discovery ](https://dev.to/nomzykush/basic-guide-to-kubernetes-service-discovery-dmd) - overview of terms and concepts related to service discovery in kubernetes
* Medium - Jul 2023 - [Service Registry and Discovery: Kubernetes Microservice Communication](https://medium.com/@josesousa8/service-registry-and-discovery-kubernetes-microservice-communication-36b314fcc06) - service registry and discovery is already part of kubernetes
* Medium - Mar 2023 - [How three lines of configuration solved our gRPC scaling issues in Kubernetes](https://medium.com/jamf-engineering/how-three-lines-of-configuration-solved-our-grpc-scaling-issues-in-kubernetes-ca1ff13f7f06) - some grpc client keep alive and pod readyness tweaking tips
* Kubernetes blog - Nov 2018 - [gRPC Load Balancing on Kubernetes without Tears](https://kubernetes.io/blog/2018/11/07/grpc-load-balancing-on-kubernetes-without-tears/) - recommended using linkerd as a grpc proxy
* gRPC blog - Jun 2017 - gRPC Load Balancing - [Recommendations and best practices](https://grpc.io/blog/grpc-load-balancing/#recommendations-and-best-practices) - gRPC Load Balancing
* grpc Name Resolution - [Name Syntax](https://github.com/grpc/grpc/blob/master/doc/naming.md)