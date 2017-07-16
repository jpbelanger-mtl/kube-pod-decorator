# kube-pod-decorator

Allows secrets from [Vault](https://vaultproject.io) and config from [Consul](https://www.consul.io/) for use with Kubernetes' Pods.


## About

This project was started to try and solve the different problem of managing secret and configuration is kubernetes Pods.

There are a few project around to allow for kubernetes to use secret from Vault. But most of these don't merge the use of Vault and Consul together. This project not only allows one pod to get Secret from Vault but also complete configuration stored in Consul. 

The idea behind this was to allow our dev team to have control over the "non secret" configuration while also being able to use secrets in those configuration files or as environment variable. 

As an example, spring-boot allows override to be pushed as Environment Variable, while the main application.yaml can contains every thing. We simply wanted dev to take ownership of their services.

The choice to select a wrapper process instead of a side-car pod came to the following criterias:

* No hack at the mais Pod's startup to "wait" for the secrets (since pods are all run at the same time).
* Able to inject environment variables, not just files.
* Move all the configuration and logic of it outside of kube. This allows us to control access to kubernetes as much as possible, while delivery pipeline only has to push to Consul. Splitting the service definition from service configuration as much as possible.
* Revoke secrets on container termination.
* Be able to run any docker image without having to build a custom one "extending" it (FROM directive)

## Features

* Receives a token as a file and use it to fetch secrets and consul configuration
* Manages Vault's token lease (with configurable TTL)
* Injects value as file or environment variable
* Supports configuration templates (with variable expansion)
* Revokes token on process termination

## Diagram
![Diagram](https://rawgithub.com/jpbelanger-mtl/kube-pod-decorator/master/diagrams/kube-pod-decorator.svg)

## Todo

* Manage each secret lease individually
* Integrate vault-controller logic within the wrapper process. Removing the need to have more than one init-container. This will also remove the need to push the token to the mounted volume.

## Usage

### Install

Compile go binary:<br/>
`make build`

Generate docker image (defaults to latest version):<br/>
`make docker`

### Injecting wrapper binary

kube-pod-decorator works with any docker image. You simply need to override the default startup binary of your container with this wrapper. To do so you first need to inject the binary inside the container. Easiest way, without rebuilding the docker image, is to push it to a volume mount from a init-container

```
apiVersion: v1
[...]
  initContainers:
  - name: init-myservice
    image: kube-pod-decorator:0.0.1
    command: ['sh', '-c', 'cp /app/kube-pod-decorator /tmp']
    volumeMounts:
    - mountPath: /tmp
      name: wrapper-volume
  volumes:
    - name: wrapper-volume
      emptyDir: {}
```

### Wrapper process
Once the wrapper binary is injected in the main pod container, we simple need to override the command call to load the wrapper instead of the app binary. The application binary and arguments are passed as to the wrapper process instead.

Example command override:

```
[...]
  containers:
  - name: myapp-container
    image: vault-demo-injection:latest
    command: ['/tmp/kube-pod-decorator', '/app/vault-demo-injection']
    env:
[...]
```

### Wrapper parameters
The wrapper process configuration is as follow:

* VAULT_ADDR : Address of the vault server
* CONSUL_HTTP_ADDR : HTTP Address of consul
* K8SPODDECORATOR_VAULTSECRETPATH : Path to this application vault token (unwrapped) defaults to `/var/run/secrets/vaultproject.io/secret.json`
* K8SPODDECORATOR_APPLICATIONNAME : Name of the application config folder in consul 
* K8SPODDECORATOR_LOGLEVEL : Log level of the wrapper (info, debug, warn)
* K8SPODDECORATOR_VAULTLEASEDURATIONSECONDS : How long the lease of the token should be in seconds (defaults to 600)
* K8SPODDECORATOR_VAULTRENEWFAILURERETRYINTERVALSECONDS : Interval in seconds between retry in case of renewal failure (defaults to 10)
* K8SPODDECORATOR_VAULTLEASERENEWALPERCENTAGE : At which percentage of the lease duration should we call for renewal (defaults to 75%)

Example configuration:

```
  containers:
  - name: myapp-container
    image: vault-demo-injection:latest
    command: ['/tmp/kube-pod-decorator', '/app/vault-demo-injection']
    env:
      - name: VAULT_ADDR
        value: http://vault:8200
      - name: K8SPODDECORATOR_VAULTSECRETPATH
        value: /tmp/vault.token
      - name: K8SPODDECORATOR_APPLICATIONNAME
        value: vault-demo-injection
      - name: CONSUL_HTTP_ADDR
        value: consul:8500
```

### Injection configuration syntax
Configuration injection is kept in consul, the configuration file is in json format and follows a syntax similar to kubernetes' secret.

```
key: vault-demo-injection
env:
  - name: DEMO_GREETING
    value: "Hello from the value in consul"
  - name: DEMO_GENERIC_SECRET
    valueFrom:
      secretKeyRef:
        name: myGenericSecret
        key: foo
  - name: DEMO_MYSQL_USERNAME
    valueFrom: 
      secretKeyRef:
        name: myDBSecret
        key: username
  - name: DEMO_MYSQL_PASSWORD
    valueFrom: 
      secretKeyRef:
        name: myDBSecret
        key: password
files:
  - name: example_config.yaml
    destination: /tmp/example_config.yaml
templates:
  - name: application.yaml
    destination: /tmp/application.yaml
    env:
      - name: DEMO_OTHER_GENERIC_SECRET
        valueFrom:
          secretKeyRef:
            name: myGenericSecret
            key: bar
      - name: CLUSTER_NAME
        valueFrom:
          consulKeyRef:
            name: myConsulValue
            key: name
vault:
  - name: myDBSecret
    type: database
    path: database/creds/readonly
  - name: myGenericSecret
    type: generic
    path: secret/foo
consul:
  - name: myConsulValue
    path: config/cluster-info
    type: json
```

* `key`: Simple reference to the application name, used as validation only (must match wrapper `ApplicationName`)
* `env`: List of environment variables to inject to the wrapped process
* `files`: List of files to mount to the volume (as-is)
* `templates`: List of templates (`go template` format) to mount to the volume and the list of variables available for expansion
* `vault`: Vault's secret path to "mount"
* `consul`: Consul value path to "mount"