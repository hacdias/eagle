---
title: Naming System
---

Naming systems are useful to allow for users (clients) to find servers. The most widely known implementation is [DNS](/notes/domain-name-system/).

- A name can be absolute or relative.

## Goal

- Associate a nasme with a resource.

## Why do we need names

- Locate resources
- Share resources
- Easier to communicate

## Concepts

- **Name**: identifies, given name by a human with semantic meaning for humans.
- **Identifier**: identifies, system controlled.
- **Address**: identifier that allows for direct access.
- **Namespace**: set of rules that define the universe of admissible names.
- **Authority**:
    - Defines the rules of naming management.
    - Must guarantee that all rules are satisfied.
    - Authority can be delegated (hierarchies).
    - Examples:
      - IP Address --> IANA
      - Ethernet --> Xerox and manufacturer
      - UUID --> IETF Standard
      - DNS --> IANA/ICANN + Delegation

## Namespace Properties

- **Uniqueness**: in a certain context, a name can only be associated with one object to avoid ambiguity. How to guarantee?
  - Central Attribution
    - High latency, single point of failure
    - Public IP addresses, Ethernet addresses
  - Local Attribution with Broadcast
    - Complicated on a large scale
    - Simple and useful in local networks

## Practical Solutions for Global Naming

### Unstructured Naming

- Can be independently attributed by any context
- Can be generated in a random or pseudorandom way
- Example: GUID, UUID

### Hierarchy Naming

- Global names composed by concatenation of local names
- Example: phone numbers, [DNS](/notes/domain-name-system/) addresses, file names

## Homogeneity and Heterogeneity

- **Homogenous**
  - Single component (e.g., Ethernet address)
  - Multiple components with the same structure and meaning (e.g., UNIX pathname)
- **Heterogenous**
  - Multiple components with structures and different meanings. E.g.:
    - Windows pathname: C:/a/b/c
    -  [URL](/notes/uri/#url): http://machine[:port]/my/dir

## Pure vs Impure Names

- **Impure**: parts of the name are used for its resolution (e.g., [URLs](/notes/uri/#url), IPs).
- **Pure**: does not locate the object, just identifies it (e.g., UUIDs, [URNs](/notes/uri/#urn)).
  - Really complicated to implement.
  - *Question:* [IPFS](/notes/ipfs/) names are pure, right? If so, they are supposed to be really difficult to impalement. But not impossible. Needs investigation and thought. I was taught that this strategy is impractical because we don't know where to start the search of the object. However, since IPFS uses a DHT and the object names are somewhat related to the user's own Peer ID, then there is a way to know where to start looking for the object. So... is it pure or not?