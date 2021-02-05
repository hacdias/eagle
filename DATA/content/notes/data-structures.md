---
math: true
title: Data Structures
---

## First In, First Out (FIFO)

**First In, First Out**, also known as **FIFO**, designates the behavior where the first element to be added to a data structure is also the first to be removed. The most common implementation of this are **queues**.

## Last In, Last Out (LIFO)

**Last In, Last Out**, also known as **LIFO**, designates the behavior where the last element to be added to a data structure is also the first to be removed. The most common implementation of this are **stacks**.

## Linked Lists

- Set of items, each with data and a pointer to the next item.
- The head of the list is a pointer to the first item.
- The end of the list is marked by a NULL pointer.

## Doubly Linked Lists

- Similar to Linked Lists but there's a pointer to the previous value too.

## Hash Tables

## Binary Trees

## Graphs

### Strongly Connected Components

#### Tarjan's Algorithm

- $O(V+E)$ complexity
- It's basically a [DFS](/notes/search-algorithms/#depth-first-search-dfs) with a different visit function.
- Needs a `low[]` matrix
- An `L` queue
- A `visited` counter

```
tarjan_visit(u):
  d[u] = low[u] = visited
  visited++
  push(L, u)
  for each v in adj[u]
    if d[v] = ∞ || v in L
      if d[v] = ∞
        tarjan_visit(v)
      low[u] = min(low[u], low[v])
  if low[u] = d[u]   # SCC root
    do
      v = pop(L)     # SCC vertices
    while v = u
```

### Directed Acyclic Graph (DAG)

### Topological Ordering

- $O(V+E)$ complexity.
- Linear ordering of a graph's (DAG) vertices such that for every edge *(u,v)*, *u* comes before *v*.
- Algorithm:
  1. Run DFS to get end times (f)
  2. When a vertex is closed, add to the beginning of a list.
  3. Return the list.