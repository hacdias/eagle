---
math: true
publishDate: "2020-09-28T09:00:00.000+02:00"
tags:
- algorithms
- mondayletter
title: A take on approximation algorithms
---

I recently started my Master's degree in Computer Science and Engineering (MSCE) and one of the first courses I am following is Advanced Algorithms. It is actually the only single course that is mandatory for all MSCE students, independently of the track/stream they chose. I chose the Web Stream.

The Advanced Algorithms course is divided into three main topics and we recently finished the first one: approximation algorithms. As a way of consolidating my knowledge and also reading a bit more than the notes given by my professor, I decided to write this essay. But first, what is an approximation algorithm and why is it important?

Some of the most well-known problems within Computer Science are virtually impossible to solve optimally at scale in a useful time. One of the most important known problems is the Traveling Salesmen problem. This problem boils down to: given a list of cities and the distances between each pair of them, what's the shortest possible route to go from a certain city to itself, visiting all of the other cities. This is problem has many more applications that its literal sense, hence its importance.

<!--more-->

<a id="continue-newsletter"></a>

Those kind of problems are called [*NP-hard*](https://en.wikipedia.org/wiki/NP-hardness) problems and there is no algorithm that solves them in polynomial time, which leaves us with exponential algorithms or worse. For small datasets, that might not be a problem. However, to solve a big problem, with a large dataset we can't afford to use slow algorithms.

Thus, there exist two ways of solving this problems even though they're not optimal:

1. Use an *heuristic*, usually by some observation such as "I noticed that this happened, so let's try for the rest", even though there's no proof of it. It may or may not give a good solution, but there's no guarantee whatsoever.
2. Use an *approximation algorithm*, which guarantees that the solution is within a certain range from the optimal solution and that can be proved.

So let's quickly say that an **optimization problem** is a problem for which we want to find the best solution from all feasible solutions. Usually, there are two different kinds of optimization problems:

- **Minimization problems**, where we the best solution refers to the solution with the lowest value; and
- **Maximization problems**, where the best solution refers to the solution with the highest values.

Before getting deeper into the subject, let's quickly define some things:

1. An algorithm $\text{ALG}$ for a minimization problem is called a $\rho$\-approximation algorithm, for some $\rho > 1$ if $\text{ALG}(I)\leq \rho \cdot \text{OPT}(I)$ for all inputs $I$.
2. An algorithm $\text{ALG}$ for a maximization problem is called $\rho$\-approximation algorithm, for some $\rho > 1$ if $\text{ALG}(I)\geq \rho \cdot \text{OPT}(I)$ for all inputs $I$.

We say that an approximation ratio $\rho$ is **tight** when $\rho=\sup_I \text{ALG}(I)/\text{OPT}(I)$, i.e., when $\rho$ is the smallest upper bound for all possible inputs $I$.

To prove that an algorithm is a $\rho$\-approximation algorithm is that the bound $\rho$ is tight, we have to show these two things:

1. Prove $\text{ALG}$ is a $\rho$-approximation algorithm, i.e., $\text{ALG}(I) \leqslant \rho \cdot \text{OPT}(I)$ for all different inputs $I$.
2. There is some input $I$ with $\text{ALG}(I)=\rho \cdot \text{OPT}(I)$.

An important step to prove this is to define lower bounds because we don't know $\text{OPT}$, the optimal solution. By setting a lower bound (or upper bound for maximization problems) on $\text{OPT}$, we can show that if our algorithm produces a solution whose value is at most a factor $\rho$ from the lower bound, then it is also within a factor $\rho$ of $\text{OPT}$.

Let's take the load balancing problem example. In this problem, we have a set $J$ jobs, each one taking $t_j$ time to complete, for all $j \in J$. The goal is to assign the jobs to $m$ machines $M_1,...,M_m$ in order to take the smallest amount of time in total (the so-called _makespan_). This is an NP-hard problem.

To solve this problem, let's use the simplest algorithm: greedy scheduling, where we assign the next job to the machine with the smallest load so far. Here's some pseudocode:

```
set the load of all machines to 0
for each job j in J do
	assign the job j to the machine with minimum load
	increase the load of the machine with the job time
end for
```

This algorithm clearly works and does what it is intended. But is it good? How close are we from the optimal solution? To argue about this, we first need to set a lower bound on the optimal solution because well... we don't actually know the optimal solution.

First of all, let's note that the best case scenario is when all machines have exactly the same load. We can define it as $\frac{1}{m}\sum_{1\leqslant j \leqslant n} t_j$. However, imagine the simple case where there are lots of small jobs and then a super large job. In that case, the lower bound would need to be as high as the biggest job. So, now we can say:

$$\text{OPT} \geqslant \max(\frac{1}{m}\sum\nolimits_{1\leqslant j \leqslant n} t_j, \max\nolimits_{1\leqslant j \leqslant n} t_j) = \text{LB}$$

Let's then prove that this algorithm is a 2-approximation algorithm. For that, we have to take a few things into consideration:

- $M_{i^*}$ is the machine that determines the makespan i.e., the machine with the highest load.
- $j^*$ is the last job assigned to $M_{i^*}$.
- $load^´(M_i)$ denotes the load of the machine $M_i$ before $j^*$ was assigned.

Knowing that, we also need to note that using this algorithm, at the time the job $j^*$ is assigned to $M_{i^*}$, $M_{i^*}$ is the machine with the lowest load. Then  $load^´(M_i^*)\leqslant load^`(M_i)$ for all $m$ machines.

$$
\begin{aligned}
load^´(M_{i^*}) &\leqslant \frac{1}{m} \cdot \sum\nolimits_{1 \leqslant i \leqslant m}load^´(M_i)\\\\
&= \frac{1}{m} \cdot \sum\nolimits_{1 \leqslant j < j^*}t_j\\\\
&< \frac{1}{m} \cdot \sum\nolimits_{1 \leqslant j \leqslant nn}t_j\\\\
&\leqslant \text{LB}
\end{aligned}
$$

Having proved that, it now follows that:

$$
\begin{aligned}
load(M_i^*) &= t_{j^*} + load'(M_{i^*})\\\\
&\leqslant t_{j^*} + LB\\\\
&\leqslant \max\nolimits_{1\leq j \leq n}t_j + \text{LB}\\\\
&\leqslant 2 \cdot \text{LB}\\\\
&\leqslant 2 \cdot \text{OPT}
\end{aligned}
$$

Now we proved that a greedy algorithm for the load balancing problem can give us an answer twice as large as the optimal one. And we proved it without even knowing the optimal solution!

I was hoping to include more things but this got longer than I expected! I may or not come to this topic again soon in the near future. Stay tuned for next week's post!