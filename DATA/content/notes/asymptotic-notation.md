---
math: true
title: Asymptotic Notation
---

- **Upper asymptotic bound:** worst-case scenario. Denoted by $O$.
- **Lower asymptotic bound:** best-case scenario. Denoted by $\Omega$.
- **Tight asymptotic bound:** denoted by $\theta$. Iff $O(g(n))=\Omega(g(n))$.

## Analysis (Master Theorem)

The **master theorem** for divide-and-conquer recurrences allows us to make an asymptotic analysis of a certain function.

**Recurrence relation:**

$T(n) = a*T(\frac{n}{b})+f(n)$, with $a>=1$ and $b>1$

Where:

- Input of size $n$
- Takes $f(n)$ time to to compute

Cases:

1. When $f(n)=O(n^{\log_b a-\epsilon}), \epsilon>0$, then $T(n)=\theta(n^{\log_b a})$.
2. When $f(n)=\theta(n^{\log_b a})$, then $T(n)=\theta(n^{\log_b a}\log n)$. 
3. When $f(n)=\Omega(n^{\log_b a+\epsilon}), \epsilon>0$ and $a*f(\frac{n}{b})<c*f(n), c<1$, then $T(n)=\theta(f(n))$. 

## Recurrence Trees

They usually end up with a geometric series:

$$\sum_{i=0}^n{r^i}=\frac{r^{i-1}-1}{r-1}$$