# Rancher Monitoring Authorization Plugin

`prometheus-auth` provides a multi-tenant implementation for Prometheus in Kubernetes.

## References

### Prometheus version supported

- [v2.4.3 and above](https://github.com/prometheus/prometheus/releases/tag/v2.4.3)

### Rancher version supported

- [v2.2.0 and above](https://github.com/rancher/rancher/releases/tag/v2.2.0)

## How to use

### Running parameters

```bash
NAME:
   prometheus-auth - Deploying in the front of Prometheus to intercept and hijack the APIs

USAGE:
   prometheus-auth [global options] command [command options] [arguments...]

VERSION:
   ...

DESCRIPTION:
   
        ##################################################################################
        ##                                      RBAC                                    ##
        ##################################################################################
        Resources                 Non-Resource URLs  Resource Names       Verbs
        ---------                 -----------------  --------------       -----
        namespaces                []                 []                   [list,watch,get]
        secrets,                  []                 []                   [list,watch,get]
        selfsubjectaccessreviews  []                 []                   [create]

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --log.json                    [optional] Log as JSON
   --log.debug                   [optional] Log debug info
   --listen-address value        [optional] Address to listening (default: ":9090")
   --proxy-url value             [optional] URL to proxy (default: "http://127.0.0.1:9999")
   --read-timeout value          [optional] Maximum duration before timing out read of the request, and closing idle connections (default: 5m0s)
   --max-connections value       [optional] Maximum number of simultaneous connections (default: 512)
   --filter-reader-labels value  [optional] Filter out the configured labels when calling '/api/v1/read'
   --help, -h                    show help
   --version, -v                 print the version

```

### Start example

```bash
prometheus-auth --log.debug --proxy-url http://127.0.0.1:9090 --listen-address :9090

```

### Metrics

`GET` - `/_/metrics` [sample](METRICS)

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