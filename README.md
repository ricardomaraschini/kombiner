# Kombiner

<img src="assets/logo/logo.png" width="100" alt="kombiner logo">

Kombiner is a component responsible for serializing pod placement proposals
generated concurrently by multiple schedulers, while ensuring fairness,
preventing starvation, respecting placement policies, and fostering cooperation
across the scheduling ecosystem.

This project proposes Placement Requests Queuing as a new native coordination
mechanism for Kubernetes scheduling, especially when multiple independent
schedulers are operating simultaneously.

The core problem addressed is that multiple schedulers can race over resources
and topology constraints, potentially invalidating each other's decisions and
leading to inefficient workload placements.

Key aspects of the proposal include:

- Placement Requests: Schedulers no longer directly bind pods. Instead, they
  create "placement requests," which are new CRD objects containing a list of
  pod-to-node assignments.
- Gang Scheduling: A single placement request carries a list of assignments and
  is governed by a policy. Two policies are proposed: "AllOrNothing" (useful
  for gang scheduling) or "Lenient" (for more flexible scheduling).
- Centralized Queue and Controller: These requests are published to a central
  queue managed by a PlacementRequest controller, responsible for validation
  and binding.
- Validation: Placement requests will be validated before binding to ensure
  native cluster-scoped and (optionally) kubelet-level constraints are met.
- Fairness and Priorities: The system aims to provide fair scheduling bandwidth
  to each scheduler based on its role, reducing placement misses and mitigating
  race conditions. This can be achieved through mechanisms like weighted queues
  and limited slots per scheduler.

## Installation

At this stage Kombiner isn't ready yet to be used in production but you can
experiment with:

```bash
$ make build-image
$ kind load docker-image kombiner:latest
$ make install
```

Please refer to the [DEPLOYING.md](DEPLOYING.md) file for more details on how
to deploy in your cluster.

## Usage

Once installed a new pod will be spawned in the `kube-system` namespace, this
pod contains both the `controller` and a `scheduler` processes. Any pod created
using the `kombiner-scheduler` scheduler will be processed by the scheduler.

For example this deployment will use the `kombiner-scheduler` scheduler:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  replicas: 10
  selector:
    matchLabels:
      app: app
  template:
    metadata:
      labels:
        app: app
    spec:
      schedulerName: kombiner-scheduler
      containers:
      - name: container
        image: nginx:1.25
```

Once the scheduling is executed you can see the PlacementRequests created by
the scheduler:

```bash
$ kubectl get placementrequests
```

When pods are deleted the corresponding PlacementRequests will be collected by
the garbage collector and removed. You can also edit the controller configuration
by editing the `controller-config` configmap in the `kube-system` namespace.

## Community, discussion, contribution, and support

You can reach the contributors of this project at:

- [Slack channel](https://kubernetes.slack.com/messages/sig-scheduling)
- [Mailing list](https://groups.google.com/forum/#!forum/kubernetes-sig-scheduling)

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).
and the [contributor's guide](CONTRIBUTING.md).

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
