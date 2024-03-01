# Proxy

The proxy service is an API-Gateway for the ownCloud Infinite Scale microservices. Every HTTP request goes through this service. Authentication, logging and other preprocessing of requests also happens here. Mechanisms like request rate limiting or intrusion prevention are **not** included in the proxy service and must be setup in front of with an external reverse proxy.

The proxy service is the only service communicating to the outside and needs therefore usual protections against DDOS, Slow Loris or other attack vectors. All other services are not exposed to the outside, but also need protective measures when it comes to distributed setups like when using container orchestration over various physical servers.

## Authentication

The following request authentication schemes are implemented:

-   Basic Auth (Only use in development, **never in production** setups!)
-   OpenID Connect
-   Signed URL
-   Public Share Token
-   Client API Keys

## Configuring Routes

The proxy handles routing to all endpoints that oCIS offers. The currently available default routes can be found [in the code](https://github.com/owncloud/oCIS/blob/master/services/proxy/pkg/config/defaults/defaultconfig.go). Changing or adding routes can be necessary when writing own oCIS extensions.

Due to the complexity when defining routes, these can only be defined in the yaml file but not via environment variables.

For _overwriting_ default routes, use the following yaml example:

```yaml
policies:
  - name: oCIS
    routes:
      - endpoint: /
        service: com.owncloud.web.web
      - endpoint: /dav/
        service: com.owncloud.web.ocdav
```

For adding _additional_ routes to the default routes use:

```yaml
additional_policies:
  - name: oCIS
    routes:
      - endpoint: /custom/endpoint
        service: com.owncloud.custom.custom
```

A route has the following configurable parameters:

```yaml
endpoint: ""       # the url that should be routed
service: ""        # the service the url should be routed to
unprotected: false # with false (default), calling the endpoint requires authorization.
                   # with true, anyone can call the endpoint without authorisation.
```

## Automatic Quota Assignments

It is possible to automatically assign a specific quota to new users depending on their role.
To do this, you need to configure a mapping between roles defined by their ID and the quota in bytes.
The assignment can only be done via a `yaml` configuration and not via environment variables.
See the following `proxy.yaml` config snippet for a configuration example.

```yaml
role_quotas:
    <role ID1>: <quota1>
    <role ID2>: <quota2>
```

## Automatic Role Assignments

When users login, they do automatically get a role assigned. The automatic role assignment can be
configured in different ways. The `PROXY_ROLE_ASSIGNMENT_DRIVER` environment variable (or the `driver`
setting in the `role_assignment` section of the configuration file) selects which mechanism to use for
the automatic role assignment.

When set to `default`, all users which do not have a role assigned at the time for the first login will
get the role 'user' assigned. (This is also the default behavior if `PROXY_ROLE_ASSIGNMENT_DRIVER`
is unset.)

When `PROXY_ROLE_ASSIGNMENT_DRIVER` is set to `oidc` the role assignment for a user will happen
based on the values of an OpenID Connect Claim of that user. The name of the OpenID Connect Claim to
be used for the role assignment can be configured via the `PROXY_ROLE_ASSIGNMENT_OIDC_CLAIM`
environment variable. It is also possible to define a mapping of claim values to role names defined
in ownCloud Infinite Scale via a `yaml` configuration. See the following `proxy.yaml` snippet for an
example.

```yaml
role_assignment:
    driver: oidc
    oidc_role_mapper:
        role_claim: ocisRoles
        role_mapping:
            - role_name: admin
              claim_value: myAdminRole
            - role_name: spaceadmin
              claim_value: mySpaceAdminRole
            - role_name: user
              claim_value: myUserRole
            - role_name: guest
              claim_value: myGuestRole
```

This would assign the role `admin` to users with the value `myAdminRole` in the claim `ocisRoles`.
The role `user` to users with the values `myUserRole` in the claims `ocisRoles` and so on.

Claim values that are not mapped to a specific ownCloud Infinite Scale role will be ignored.

Note: An ownCloud Infinite Scale user can only have a single role assigned. If the configured
`role_mapping` and a user's claim values result in multiple possible roles for a user, the order in
which the role mappings are defined in the configuration is important. The first role in the
`role_mappings` where the `claim_value` matches a value from the user's roles claim will be assigned
to the user. So if e.g. a user's `ocisRoles` claim has the values `myUserRole` and
`mySpaceAdminRole` that user will get the oCIS role `spaceadmin` assigned (because `spaceadmin`
appears before `user` in the above sample configuration).

If a user's claim values don't match any of the configured role mappings an error will be logged and
the user will not be able to log in.

The default `role_claim` (or `PROXY_ROLE_ASSIGNMENT_OIDC_CLAIM`) is `roles`. The default `role_mapping` is:

```yaml
- role_name: admin
  claim_value: ocisAdmin
- role_name: spaceadmin
  claim_value: ocisSpaceAdmin
- role_name: user
  claim_value: ocisUser
- role_name: guest
  claim_value: ocisGuest
```

## Recommendations for Production Deployments

In a production deployment, you want to have basic authentication (`PROXY_ENABLE_BASIC_AUTH`) disabled which is the default state. You also want to set up a firewall to only allow requests to the proxy service or the reverse proxy if you have one. Requests to the other services should be blocked by the firewall.

## Caching

The `proxy` service can use a configured store via `PROXY_OIDC_USERINFO_CACHE_STORE`. Possible stores are:
  -   `memory`: Basic in-memory store and the default.
  -   `redis-sentinel`: Stores data in a configured Redis Sentinel cluster.
  -   `nats-js-kv`: Stores data using key-value-store feature of [nats jetstream](https://docs.nats.io/nats-concepts/jetstream/key-value-store)
  -   `noop`: Stores nothing. Useful for testing. Not recommended in production environments.
  -   `ocmem`: Advanced in-memory store allowing max size. (deprecated)
  -   `redis`: Stores data in a configured Redis cluster. (deprecated)
  -   `etcd`: Stores data in a configured etcd cluster. (deprecated)
  -   `nats-js`: Stores data using object-store feature of [nats jetstream](https://docs.nats.io/nats-concepts/jetstream/obj_store) (deprecated)

Other store types may work but are not supported currently.

Note: The service can only be scaled if not using `memory` store and the stores are configured identically over all instances!

Note that if you have used one of the deprecated stores, you should reconfigure to one of the supported ones as the deprecated stores will be removed in a later version.

Store specific notes:
  -   When using `redis-sentinel`, the Redis master to use is configured via e.g. `OCIS_CACHE_STORE_NODES` in the form of `<sentinel-host>:<sentinel-port>/<redis-master>` like `10.10.0.200:26379/mymaster`.
  -   When using `nats-js-kv` it is recommended to set `OCIS_CACHE_STORE_NODES` to the same value as `OCIS_EVENTS_ENDPOINT`. That way the cache uses the same nats instance as the event bus.
  -   When using the `nats-js-kv` store, it is possible to set `OCIS_CACHE_DISABLE_PERSISTENCE` to instruct nats to not persist cache data on disc.


## Presigned Urls

To authenticate presigned URLs the proxy service needs to read signing keys from a store that is populated by the ocs service. Possible stores are:
  -   `nats-js-kv`: Stores data using key-value-store feature of [nats jetstream](https://docs.nats.io/nats-concepts/jetstream/key-value-store)
  -   `redis-sentinel`: Stores data in a configured Redis Sentinel cluster.
  -   `ocisstoreservice`:  Stores data in the legacy oCIS store service. Requires setting `PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_NODES` to `com.owncloud.api.store`.

The `memory` or `ocmem` stores cannot be used as they do not share the memory from the ocs service signing key memory store, even in a single process.

Make sure to configure the same store in the ocs service.

Store specific notes:
  -   When using `redis-sentinel`, the Redis master to use is configured via e.g. `OCIS_CACHE_STORE_NODES` in the form of `<sentinel-host>:<sentinel-port>/<redis-master>` like `10.10.0.200:26379/mymaster`.
  -   When using `nats-js-kv` it is recommended to set `OCS_PRESIGNEDURL_SIGNING_KEYS_STORE_NODES` to the same value as `PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_NODES`. That way the ocs uses the same nats instance as the proxy service.
  -   When using the `nats-js-kv` store, it is possible to set `PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_DISABLE_PERSISTENCE` to instruct nats to not persist signing key data on disc.
  -   When using `ocisstoreservice` the `PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_NODES` must be set to the service name `com.owncloud.api.store`. It does not support TTL and stores the presigning keys indefinitely. Also, the store service needs to be started.


## Special Settings

When using the oCIS IDP service instead of an external IDP:

-   Use the environment variable `OCIS_URL` to define how oCIS can be accessed, mandatory use `https` as protocol for the URL.
-   If no reverse proxy is set up, the `PROXY_TLS` environment variable **must** be set to `true` because the embedded `libreConnect` shipped with the IDP service has a hard check if the connection is on TLS and uses the HTTPS protocol. If this mismatches, an error will be logged and no connection from the client can be established.
-   `PROXY_TLS` **can** be set to `false` if a reverse proxy is used and the https connection is terminated at the reverse proxy. When setting to `false`, the communication between the reverse proxy and oCIS is not secured. If set to `true`, you must provide certificates.

## Metrics

The proxy service in oCIS has the ability to expose metrics in the prometheus format. The metrics are exposed on the `/metrics` endpoint. There are two ways to run the oCIS proxy service which has an impact on the number of metrics exposed.

### 1) Single Process Mode
In the single process mode, all oCIS services are running inside a single process. This is the default mode when using the `oCIS server` command to start the services. In this mode, the proxy service exposes metrics about the proxy service itself and about the oCIS services it is proxying. This is due to the nature of the prometheus registry which is a singleton. The metrics exposed by the proxy service itself are prefixed with `ocis_proxy_` and the metrics exposed by other oCIS services are prefixed with `ocis_<service-name>_`.

### 2) Standalone Mode
In this mode, the proxy service only exposes its own metrics. The metrics of the other oCIS services are exposed on their own metrics endpoints.

### Available Metrics
The following metrics are exposed by the proxy service:

| Metric Name                      | Description                                                                                                                                                                                                                   | Labels                                |
|----------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------|
| `ocis_proxy_requests_total`      | [Counter](https://prometheus.io/docs/tutorials/understanding_metric_types/#counter) metric which reports the total number of HTTP requests.                                                                                   | `method`: HTTP method of the request  |
| `ocis_proxy_errors_total`        | [Counter](https://prometheus.io/docs/tutorials/understanding_metric_types/#counter) metric which reports the total number of HTTP requests which have failed. That counts all response codes >= 500                           | `method`: HTTP method of the request  |
| `ocis_proxy_duration_seconds`    | [Histogram](https://prometheus.io/docs/tutorials/understanding_metric_types/#histogram) of the time (in seconds) each request took. A histogram metric uses buckets to count the number of events that fall into each bucket. | `method`: HTTP method of the request  |
| `ocis_proxy_build_info{version}` | A metric with a constant `1` value labeled by version, exposing the version of the oCIS proxy service.                                                                                                                        | `version`: Build version of the proxy |

### Prometheus Configuration
The following is an example prometheus configuration for the single process mode. It assumes that the proxy debug address is configured to bind on all interfaces `PROXY_DEBUG_ADDR=0.0.0.0:9205` and that the proxy is available via the `oCIS` service name (typically in docker-compose). The prometheus service detects the `/metrics` endpoint automatically and scrapes it every 15 seconds.

```yaml
global:
  scrape_interval: 15s
scrape_configs:
  - job_name: ocis_proxy
    static_configs:
    - targets: ["oCIS:9205"]
```
