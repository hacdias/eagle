---
tags:
- ipfs
title: IPFS Whitepaper Draft 3
---

- **paper**: https://github.com/ipfs/papers/raw/master/ipfs-cap2pfs/ipfs-p2p-file-system.pdf

> Note: this paper is quite old and [IPFS spec](https://github.com/ipfs/specs) has evolved a lot since then so some of these information might not be up to date. However, it is always interesting to understand what changed and why.

## Introduction

1. There's no file system nowadays that provides a global, low latency and decentralized distribution.
2. No current protocol (like [HTTP](/notes/http/)) uses the new file distribution techniques invented in the past 15 years.
3. **Goal**: enhance the web we know nowadays without degrading UX.
4. Explore how the Merkle DAG can be used for high-throughput oriented file systems.

## Background

### Distributed Hash Tables

IPFS uses a [DHT](Distributed%20Hash%20Table) to locate which peers have what content. In common implementations, DHTs would serve as a place to store the data directly. In this case, it stores the peer that has the content.

### Block Exchange

IPFS uses [BitSwap](BitSwap), a [BitTorrent](BitTorrent) inspired data exchange protocol.

### Objects

- Objects are content addresses just like in [Git](Git) by their cryptographic hash.
- Links to other objects are embedded, forming a Merkle DAG (**it now uses the [IPLD](IPLD) instead!!!**).

### Self-Certified File Systems

File systems in which their address location self certufues the server.

## Design

- No nodes are privileged
- Objects are stored in local storage
- Objects can represent files or other data structures

**Stack:**

- **Identities**
  - each node has a public key
  - peer id = hash(publicKey)
  - public and private key stored encrypted with a passphrase
  - generation based on S/Kademlia
- **Network**
  - uses [libp2p](https://libp2p.io/) hence can use any transport protocol
  - reliability on unreliable protocols
- **Routing**
  - uses a dht based on [Kademlia (DHT)](Kademlia%20(DHT)) and [Coral (DHT)](Coral%20(DHT))
  - if value <= 1KB, then stored directly on dht
  - otherwise peer id stored
- **Exchange**
  - blocks can be shared between objects
  - uses [BitSwap](BitSwap):
    - each peer has a wantlist and a havelist
    - maximize the trade performance for the node
    - prevent exploitation from freeloaders
- **Objects**
  - Uses [IPLD](IPLD)
  - Deduplication of blocks
  - Content-addressed
  - Can use different storage backends
  - Pinning --> make sure an object isn't removed
  - Anyone can publish objects in the DHT
- **Files**
  - `blob`: addressable unit of data, represents  a file.
  - `list`: represents a file composed by other objects, contain a sequence of `blocks` or `lists`.
  - `tree`: represents a directory, maps names to hashes.
  - `commit`: snapshot in the version history of any object. (_is this still up to date?_)
  - splitting files into lists
- **Naming**
  - See [IPNS](IPNS)
- (OPINION) "Using IPFS" section: it's a bit sad to see that most of the points are yet to be feasible nowadays. However, the protocol has evolved a lot in the past months and years.