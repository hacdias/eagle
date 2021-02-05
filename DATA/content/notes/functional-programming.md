---
math: true
title: Functional Programming
---

Functional Programming is a [Programming Paradigm](/notes/programming-fundamentals/#programming-paradigms) in which programs are mostly built by applying and composing functions.

## Reducer

A **reducer** is a function that iterates over a list, applying a certain operation to all elements. On each call, it sends the previous result as well as the next element on the list. Only the last result is preserved.

**Iterative**:

```python
def reduce(fn, lst):
  res = lst[0]
  for x in lst[1:]:
    res = fn(res, x)
  return res
```

**Recursive**:

```python
def reduce(fn , lst):
  if len(lst) == 1:
    return lst[0]
  else:
    return fn(lst[0], reduce(fn, lst[1:]))
```

## Filter

A **filter** receives a predicate and a list and then it checks for the elements that satisfy the predicate. Only those are returned.

**Iterative**:

```python
def filter(fn, lst):
  res = []
  for x in lst:
    if fn(x):
      res = res + [x]
  return res
```

**Recursive**:

```python
def filter(fn, lst):
  if lst == []:
    return lst
  elif fn(lst[0]):
    return [lst[0]] + filter(fn, lst[1:])
  else:
    return filter(fn, lst[1:])
```

## Map

A **map** transforms a certain list by applying the same operation to all elements.

**Iterative**:

```python
def map(fn, lst):
  res = list()
  for e in lst:
    res = res + [fn(e)]
  return res
```

**Recursive**:

```python
def map(fn, lst):
  if lst == []:
    return lst
  else:
    return [fn(lst[0])] + map(fn, lst[1:])
```