# kube-timer: get the provision/deletion time of kubernetes resources easily

## Install

```shell
go install github.com/nilo19/kube-timer
kube-timer -h
```

## Usage

```shell
# Get the provision time of a LoadBalancer service.
kube-timer svc -f <service definition yaml file>

# Get the provision time of multiple LoadBalancer services (create the next service after the previous one is ready).
kube-timer svc -f <service definition yaml file> -c <number of services>

# Get the provision time of multiple LoadBalancer services (create all services and wait for them to be ready).
kube-timer svc -f <service definition yaml file> -c <number of services> -a

# Get the provision time of multiple LoadBalancer services and use event logs to calculate.
kube-timer svc -f <service definition yaml file> -c <number of services> -a --started-event-reason <reason of an event> --finished-event-reason <reason of an event>

# Get the deletion time of a LoadBalancer service.
kube-timer svc -d -n <service name> --namespace <service namespace> --started-event-reason <reason of an event> --finished-event-reason <reason of an event>

# Get the deletion times of all LoadBalancer services in the cluster.
kube-timer svc -D -n <service name> --namespace <service namespace> --started-event-reason <reason of an event> --finished-event-reason <reason of an event>
```
