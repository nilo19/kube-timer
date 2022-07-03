---
title: Instruction for kube-timer
description: Learn how to use kube-timer to calculate the provision/deletion time of Kubernetes resources.
author: CocoWang-wql
date: 07/3/2022
---

# Use kube-timer to get the provision/deletion time of kubernetes resources

This articles provides the instructions about how get the provision/deletion time of kubernetes resources. It contains the installtion steps, description of parameters and samples outputs.

## How to install kube-timer

You could run the below command in the terminal to install kube-timer tool.

```shell
go install github.com/nilo19/kube-timer
```

After the installtion, run `kube-timer -h` to check it.

## Parameters of kube-timer
| Parameter | Description |
| --------- | ------------- |
| -f | The yaml file for resources provision | 
| -c | The number of resource | 
| -a | Create all the resources together and wait for the previous resouces to be ready. The time of single resource creation would contain waiting in queue time. Without this parameter, the resources are created one by one; the next resource will only be created once the previous one is ready.|
| -d | Delete the resource |
| -n | Name of the resource to be leleted |
| -D | Delete all the resources together | 
| --started-event-reason | The reason of event. It can be used as the start time of provision/deletion |
| --finished-event-reason | The reason of event. It can be used as the end time of provision/deletion |
| --namespace | The namespace of resources | 

## Samples for the kube-timer

### Get the provision time of single LoadBalancer service.
1. Create a file named `svc.yaml` 

2. Sample YAML definition:

```yml
kind: Service
apiVersion: v1
metadata:
  Name: testsvc
  namespace: default
spec:
  selector:
    app: nsg-test
  ports:
    - name: port1
      port: 81
      targetPort: 81
      protocol: TCP
  type: LoadBalancer
  
```

3. Get the provision time of this LoadBalancer service.
```shell
kube-timer svc -f <service definition yaml file>
```

### Get the provision time of multiple LoadBalancer services.

#### Do not contain waiting-in-queue time

As we do not specify `-a`, all the services will be created one by one. The next service is only be created once the previous one is ready.
1. Create a file named `svc.yaml` 

2. Sample YAML definition:

```yml
kind: Service
apiVersion: v1
metadata:
  generateName: lb-svc-internal- # All service names are prefixed with lb-svc-internal-
  namespace: default
  # annotations:
  #   service.beta.kubernetes.io/azure-pip-tags: "a=b,c=d"
  # service.beta.kubernetes.io/azure-load-balancer-internal: "true"
  #   service.beta.kubernetes.io/azure-load-balancer-internal-subnet: "lb-subnet"
  #   service.beta.kubernetes.io/azure-load-balancer-mixed-protocols: "true"
  #   service.beta.kubernetes.io/azure-load-balancer-mode: "test-capz-2-vmss-0"
  #   service.beta.kubernetes.io/azure-shared-securityrule: "true"
  #   service.beta.kubernetes.io/azure-deny-all-except-load-balancer-source-ranges: "true"
  #   service.beta.kubernetes.io/azure-dns-label-name: "test-dns-lb-svc-1-managed"
spec:
  selector:
    app: nsg-test
  ports:
    - name: port1
      port: 81
      targetPort: 81
      protocol: TCP
    # - name: port2
    #   port: 82
    #   targetPort: 82
    #   protocol: UDP
  type: LoadBalancer
  # loadBalancerIP: 20.252.52.147
  # ipFamilies:
  # - IPv4
  # - IPv6
```

3. Get the provision time of each service and total time for the whole provision.

```shell
kube-timer svc -f <service definition yaml file> -c <number of services>
```

4.Sample output:
![image](https://user-images.githubusercontent.com/45681473/177050104-b3a7fa1e-08ad-4811-a1b2-fc6c83e15101.png)


#### Contains waiting-in-queue time
As we specify `-a` parameter here, all the services privision requests will be created together. The next service will always wait untile the previous one is ready. Thus, the provision time for signle service would contain waiting time.

```shell
kube-timer svc -f <service definition yaml file> -c <number of services> -a
```

Sample output:
![image](https://user-images.githubusercontent.com/45681473/177050241-dfdbb5b4-05ba-45d8-b829-c1e33323c07c.png)


### Get the provision time of multiple LoadBalancer services and use event logs to calculate.
By default, the provision time is calculated by detecting IP creation. With `--started-event-reason` and `--finished-event-reason`, the time would be calculated by event logs.

```shell
kube-timer svc -f <service definition yaml file> -c <number of services> -a --started-event-reason '<reason of an event>' --finished-event-reason '<reason of an event>'
```

Sample command:
```shell
kube-timer svc -f svc.yaml -c 10  -a --started-event-reason 'EnsuringLoadBalancer' --finished-event-reason 'EnsuredLoadBalancer'
```

### Get the deletion time of a LoadBalancer service.
```shell
kube-timer svc -d -n <service name> --namespace <service namespace> --started-event-reason '<reason of an event>' --finished-event-reason '<reason of an event>'
```

### Get the deletion times of all LoadBalancer services in the cluster.
```shell
kube-timer svc -D -n <service name> --namespace <service namespace --started-event-reason '<reason of an event>' --finished-event-reason '<reason of an event>'
```




