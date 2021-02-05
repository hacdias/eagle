---
math: true
title: Distributed Hash Table (DHT)
---

- **tags**: [Distributed Systems](/notes/distributed-systems/)

DHTs are Distributed Hash Tables which are P2P algorithms.

>  "A system where the nodes organize themselves in a structured overlay and establish a small amount of routing information for quick and efficient routing to other overlay nodes." - [Peer-to-Peer Systems and Applications](/notes/peer-to-peer-systems-and-applications/)

- P2P Algorithms: different implementations
- Nodes store index info about other resources
- Flat architecture: no special nodes
- Usually can find resources on $O(\log N)$
- By distributing identifiers of nodes and data equally thorough the system, the load shoud be balanced across all peers
	- PROBLEM: obviously there are some resources that are always more accessed than others, creating possibily huge differences.
- Data storage:
	- **Direct storage**: data is copied upon insertion to the responsible node.
		- Good because the data is directly on the peer.
		- Bad for bandwidth and resources.
	- **Referenced storage**: references pointers to the actual location of the data.
		- Good because there's less load on the DHT.
		- Bad becauser data is only available while the node is available.
- Can be interpreted as:
	- Routing systems
	- Storage systems
- Challenges:
	- Routing efficiency
	- Management overhead
	- Dynamics

## Algorithms

- [Chord (DHT)](/notes/chord-dht/)
- [Content Addressable Network (DHT)](/notes/content-addressable-network-dht/)
- [Tapestry (DHT)](/notes/tapestry-dht/)
- [Pastry (DHT)](/notes/pastry-dht/)
- [Kademlia (DHT)](/notes/kademlia-dht/)

### Comparison

|System|Routing Hops|Node State|Arrival|Departure|
|---|---|---|---|---|
|[Chord (DHT)](/notes/chord-dht/)|$O(\frac{1}{2}\log_2(N))$|$O(2\log_2(N))$|$O(log_2^2(N))$|$O(\log_2^2(N))$|
|[Pastry (DHT)](/notes/pastry-dht/)|$O(\frac{1}{2}\log_2(N))$|$O(\frac{1}{b}(2^b-1)\log_2(N))$|$O(log_{2^b}(N))$|$O(\log_b(N))$|
|[Content Addressable Network (DHT)](/notes/content-addressable-network-dht/)|$O(\frac{D}{2}N^{1/D})$|$O(2D)$|$O(\frac{D}{2}N^{1/D})$|$O(2D)$|
|Symphony|$O(\frac{c}{k}\log_2(N))$|$O(2k+2)$|$O(\log^2(N))$|$O(\log^2(N))$|
|Viceroy|$O(\frac{c}{k}\log_2(N))$|$O(2k+2)$|$O(\log_2(N))$|$O(\log_2(N))$|
|[Kademlia (DHT)](/notes/kademlia-dht/)|$O(\log_b(N))$|$O(b\log_b(N))$|$O(\log_b(N))$|$O(\log_b(N))$|