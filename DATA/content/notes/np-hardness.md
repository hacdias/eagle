---
title: NP-hardness
---

Some of the most well-known problems within Computer Science are virtually impossible to solve optimally at scale in a useful time. One of the most important known problems is the Traveling Salesmen problem. This problem boils down to: given a list of cities and the distances between each pair of them, what's the shortest possible route to go from a certain city to itself, visiting all of the other cities. This is problem has many more applications that its literal sense, hence its importance.

Those kind of problems are called [*NP-hard*](https://en.wikipedia.org/wiki/NP-hardness) problems and there is no algorithm that solves them in polynomial time, which leaves us with exponential algorithms or worse. For small datasets, that might not be a problem. However, to solve a big problem, with a large dataset we can't afford to use slow algorithms.

Thus, there exist two ways of solving this problems even though they're not optimal:

1. Use an *heuristic*, usually by some observation such as "I noticed that this happened, so let's try for the rest", even though there's no proof of it. It may or may not give a good solution, but there's no guarantee whatsoever.
2. Use an [Approximation Algorithms](/notes/approximation-algorithms/), which guarantees that the solution is within a certain range from the optimal solution and that can be proved.