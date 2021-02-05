---
math: true
title: Kademlia (DHT)
---

Kademlia is one implementation of a [Distributed Hash Table](/notes/distributed-hash-table/). 

The **3 parameters**:

- **Address space**: a way to uniquely identify all the peers in a network.
- A **metric** to order the peers in the address space and therefore be able to visualize them along an ordered line.
- A **projection** that will take a record key and calculate a position in the address space where the peer(s) most ideally suited to store the record should be near.

More details:

- Peers can join (or leave) at any time hence it's unstable.
- Each peer keeps links to the peers located at $2n$ of distance.
- For each multiple of 2, each peer keeps up to $K$ links.
- $K$ is determined based on the observed average churn in the network and the frequency with which the network republishes information.
	- Computed to maximize the probability of keeping the network connected while maintaining good latency values for queries.

This will:

- Allow to search the network as if it was a sorted list.
- Allows for a lookup time of $O(\log(N))$ where $N$ is the size of the network.

## Resources

- https://en.wikipedia.org/wiki/Kademlia
- [IPFS](/notes/ipfs/) DHT: https://blog.ipfs.io/2020-07-20-dht-deep-dive/