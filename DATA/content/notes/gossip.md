---
math: true
title: Gossip
---

The **Gossip architecture** is a [weak consistency](/notes/replication/) replication technique where clients send requests to the nearest replica and then the replicas propagate the information to each other periodically just like a normal gossip. It allows for high availability and tolerance to net partitions (A+P).

## Algorithm

The algorithm for this architecture can be adapted to the needs of each implementation. It does not need to be rigorously as described bellow. From now on, we will assume there are $n$ replicas.

### The Timestamp

There is the concept of **timestamp**, which is a vector with $n$ entries, where each entry symbolizes the current state of the $i$th replica. Let's define the merge operation:

```
merge(tsA, tsB):
  for each entry i in tsA
    if tsA[i] > tsB[i]
      tsB[i] = tsA[i]
```

On each request, the client sends the `prev` timestamp (from the last request). When a replica replies to the client, it returns the data, as well as their own timestamp `new`. In this case, the client executes `merge(new, prev)`.

### Replica State

Each replica contains the following state:

- _Value_: the value that is stored on each replica we want to keep in sync.
- _Value Timestamp_: a timestamp representing the operations executed to achieve the current _value_.
- _Log_: all the update operations the replica has accepted so far. It may contain already processed operations that are yet required to be propagated to other replicas.
- _Replica Timestamp_: represents all the operations the current replica has accepted, i.e., that were placed in the log.
- _Executed Operations_: a list with the unique IDs of the executed operations on the current replica.
- _Timestamp Table_: a list with the the known replica timestamps from other replicas.

### Get Operations

When a client makes a read request, it sends the `prev` timestamp as well as the request information. Then, the server proceeds as follows:

1. Compares the `prev` timestamp with the `valueTimestamp` to see if we can safely read the value to ensure consistency, i.e., if `prev` <= `valueTimestamp`.
2. If the previous condition is met, the server replies the required information.

Otherwise, wait.

### Update Operations

When a client wants to make an update, it generates a UUID _id_ to uniquely identify the request. Then, it sends the `id`, the `data` and `prev`, When the request `req` arrives in the server, the replica checks decides whether of not to discard the request. A request is discarded if:

- The `id` is present on the executed operations list; or
- The `id` is present on any log record.

If the request is accepted, then we follow the algorithm:

1. Update the replica timestamp by incrementing the $i$th entry, where `i` is the current instance number starting on 0.
2. Create a unique `timestamp` to represent the operation from now on. This timestamp is created by duplicating `prev` and replacing the ith entry with the previously calculated value.
3. Creates a new log record with the new `timestamp`, the current instance number, the `id` and the data to add.
4. Returns the new timestamp to the client.
5. Checks if the operation can be executed immediately by checking if `prev` <= `valueTimestamp`. If possible, execute `merge(timestamp, valueTimestamp)`.

Otherwise, wait.

> **Note**: if the system's data has no casual dependencies, there is no need to wait to have the updated data to apply the changes. In that specific case, most of the logic above can be simplified in order to just avoid duplications, but the updates can be immediately applied.

### Gossip Operations

This architecture requires each replica to *gossip* every $t$ time to each other in order to keep consistency. On a gossip request, a replica _i_ sends to the replica _j_:

- The instance number _i_;
- The `sourceTimestamp`, which is the replica timestamp of replica _i_;
- The logs records we estimate the replica _j_ does not have.
  
When the replica _j_ receives the gossip message from the replica _i_, it then proceeds as follows:

1. Updates the entry _i_ in the timestamp table with the `replicaTimestamp`;
2. For each log record `r`:
    - Checks if `r.id` is on the executed operations list. If so, discards.
    - Checks if `r.timestamp` > `replicaTimestamp`. If not, discards.
    - Adds the record to its own record log.
3. Updates `replicaTimestamp`, by executing `merge(sourceTimestamp, replicaTimestamp)`.
4. Goes through the log and executes all stable operations.
5. Finally, cleans up the log.

## Notes and Advices

- This is an architecture, not a protocol.
- Each use-case can be implemented with its specifics.
- There are ways to ensure the client does not get inconsistent results if they switch replicas.