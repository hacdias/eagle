---
math: true
tags:
- distributed
title: Replication
---

**Data replication** happens when the same data is stored on multiple storage devices that might not be on the same network.

## Benefits

- High **availability** since the system is still available even when some nodes are offline.
- Higher **performance** and **scalability** since the clients can connect to their closest data source. 

## Requisites

- Transparency: the client needs to see one system instead of cluster of collaborating systems.
- Consistency: the ideal model is where a client makes an update and all observers would see it immediately.

## Faults

Replication can tolerate a maximum number of [faults](/notes/fault-tolerance/):

- **Fail-silent faults**: for $f+1$ nodes, we can tolerate $f$ faults since we only need one replica to get the value.
- **Byzantine faults**: for $2f+1$ nodes, we can tolerate $f$ faults. This way we can assure that we receive $f+1$ equal responses.

## Active vs Passive

There are two types of data replication:

- **Active**, where all servers execute all requests.
- **Passive**, where there is a primary server and multiple secondary ones. Only the primary server executes the requests.

## CAP Theorem

> Building reliable distributed systems at a worldwide scale demands trade-offs between consistency and availability - Werner Vogels[^1]

Data Consistency, High Availability, Tolerance to Network Partition: only 2 of these can be achieved at a given time.

- C+A
  - Data Consistency + High Availability
  - Not tolerant to network partitions
  - Usually uses transaction protocol
  - Client and storage systems must be part of the same environment
- On large scale distributed systems, network partitions are a given, so it's highly unlikely that the C+A "paradigm" can be applied.
- C+P
  - System may not be available
- A+P
  - System may not be consistent
- ACID (Atomicity, Consistency, Isolation, Durability)
- From the practical POV, monotonic reads and read-your-writes are the most desirable properties, but not always required.

## Consistency Models

For these models, we will take into account these components:

- A storage system.
- A process A that reads and writes.
- Two other processes B and C.

### Strong Consistency

After A makes an update, both B and C will return the updated value.

**Implementations**:

- [Primary-backup](/notes/primary-backup/)
- [State Machine Replication](/notes/state-machine-replication/)

### Weak Consistency

After an update, there is no guarantee that subsequent updates will return the updated value for none of the clients, not even for the one who made the update.

**Implementations**:

- [Gossip](/notes/gossip/) architecture.

### Eventual Consistency

It's a [#Weak Consistency](#weak-consistency) model. If no new updates are made to a given item, eventually all accesses to that item will return the updated value.

Conditions such as propagation delays and others can be taken into account to calculate the *inconsistency window*.

E.g.: [Domain Name System](/notes/domain-name-system/)

### Causal Consistency

In this model, all updates that are causally related must be applied in order on all processes. Other updates might be applied at any time.

### Read-your-writes Consistency

After a process A writes an update to a given item, all subsequent reads by the same process should return the updated value.

### Session Consistency

Practical version of [#Read-your-writes Consistency](#read-your-writes-consistency). There is a session between a process and a storage system. While the session is on, the process should always read values consistent to their previous updates.

### Monotonic Read Consistency

If the process A sees an update for a given item, all subsequent reads by that process on that item must reflect that update.

### Monotonic Write Consistency

The updates made by a process A to a given item are all applied in order, i.e., each update is executed after the previous one is executed.

[^1]: Vogels, W. (2008). Eventually consistent. Queue, 6(6), 14–19. https://doi.org/10.1145/1466443.1466448