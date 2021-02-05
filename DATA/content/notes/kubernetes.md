---
title: Kubernetes
---

Kubernetes has the following main components:
- One or more master nodes
	- API server
	- Scheduler
	- Controller managers
	- etcd
- One or more worker nodes
- Distributed key-value store, such as etcd.
- Pods: group of containers.
	- Atomic unit of work in a kubernetes cluster.
- Labels: `-l`