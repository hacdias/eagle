---
title: State Machine Replication
---

**State Machine Replication** is a [Replication](/notes/replication/) technique that allows for strong consistency and high availability (C+A), hence it does not tolerate net partitions.

- It's a **generic** solution for [Fault Tolerant](/notes/fault-tolerance/) services.
- Each server is a state machine defined by state variables.
- The operations are atomic.

Every server is required to have the same initial state and agreement (interface), to execute the operations by the same order and to have deterministic operations.

There are many algorithms that implement this technique:

- Fail-silent Faults
  - Paxos
  - [RAFT](/notes/raft/)
- Byzantine Faults
  - PBFT