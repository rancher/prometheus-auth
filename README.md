# Rancher Monitoring Authorization Plugin

`prometheus-auth` provides a multi-tenant implementation for Prometheus in Kubernetes.

## References

### Prometheus version supported

- [v2.4.3 and above](https://github.com/prometheus/prometheus/releases/tag/v2.4.3)

### Rancher version supported

- [v2.1.0 and above](https://github.com/rancher/rancher/releases/tag/v2.1.0)

## How to use

### Running parameters

```bash
$ prometheus-auth -h
NAME:
   prometheus-auth - Authorization plugins for Rancher monitoring

USAGE:
   prometheus-auth [global options] command [command options] [arguments...]

...

COMMANDS:
     remote   An authorization remote reader for Prometheus, with RBAC: [namespaces](list, watch, get)
     agent    An authorization agent before Prometheus, with RBAC: [namespaces, secrets, serviceaccounts, clusterrole, clusterrolebindings, role, rolebindings](list, watch, get)
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --log.json     [optional] Log as JSON
   --log.debug    [optional] Log debug info
   --help, -h     show help
   --version, -v  print the version


```

#### Start a remote reader

```bash
$ prometheus-auth remote start -h
NAME:
   prometheus-auth remote start - Start a Prometheus remote reader

USAGE:
   prometheus-auth remote start [command options] [arguments...]

OPTIONS:
   --listen.address value              [optional] Address to listen (default: ":9201")
   --remote.read-url value             [optional] URL to read from remote (default: "http://prometheus-operated.cattle-prometheus:9090/api/v1/read")
   --remote.read-timeout value         [optional] Timeout to read from remote (default: 5s)
   --filter.external-labels-key value  [optional] Filter out the keys of configured 'externalLabels' before reading remote (default: "prometheus", "prometheus_replica")
   --filter.only                       [optional] Only filter out the setting keys, but not authorization
   --watch.resync-period value         [optional] Resync period of Kubernetes watching (default: 15m0s)
```

#### Start a agent 

```bash
$ prometheus-auth agent start -h
NAME:
   prometheus-auth agent start - Start a Prometheus agent

USAGE:
   prometheus-auth agent start [command options] [arguments...]

OPTIONS:
   --listen.address value         [optional] Address to listening (default: ":9090")
   --agent.proxy-url value        [optional] URL to proxy (default: "http://localhost:9999")
   --agent.read-timeout value     [optional] Maximum duration before timing out read of the request, and closing idle connections (default: 5m0s)
   --agent.max-connections value  [optional] Maximum number of simultaneous connections (default: 512)
   --watch.resync-period value    [optional] Resync period of Kubernetes watching (default: 15m0s)

```

# License

Copyright (c) 2014-2018 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.