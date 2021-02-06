---
math: true
title: Chord (DHT)
---

Chord is a [Distributed Hash Table](/notes/distributed-hash-table/) (DHT) protocol and algorithm. It is one of the four original DHT protocols, along with [CAN](/notes/content-addressable-network-dht/), [Tapestry (DHT)](/notes/tapestry-dht/) and [Pastry (DHT)](/notes/pastry-dht/).

- Nodes and keys are assigned an _m_-bit identifier using an [Hashing Algorithm](/notes/hashing-algorithm/). The nodes and keys are distributed across the same identifier space.
- The nodes are theoretically arranged as a circle with at most $2^m$ nodes, starting from 0 to $2^m-1$.
- Each node keeps a finger (routing) table with up to $m$ entries. The $i^{th}$ entry should contain the $(n+2^{i-1}) \mod 2^m$. If the node does not exist, it should then contain the next available successor.
    - For example, in a network with a key-size 4 (unrealistic), there are, at most 16 nodes. Each node stores the addresses for 4 nodes that are located 1, 2, 4 and 8 positions away from them, unless they do not exust.
    - This method allows for an efficient trade-off between low-amount of stored keys and fast search.
- To add a new file to the network:
  - Hash file -> Key $k$.
  - Node $k$ should store resource with key $Pk$.
  - If node $k$ does not exist, store on the next successor.
- To search for data:
  - Knowing the key $k$.
  - Query the closest node to $k$ available in our routing table.
- When a new node $y$ joins the network, the node that was storing $y$'s keys needs to be updated.
- When a node $y$ leaves the network, the next available node managed keys' must be updated.

## References

- [ITS413, Lecture 21, 29 Mar 2012](https://www.youtube.com/watch?v=qqv4OJ5Lc4E)