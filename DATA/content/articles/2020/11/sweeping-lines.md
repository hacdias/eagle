---
publishDate: "2020-11-16T09:00:00.000+01:00"
tags:
- mondayletter
- algorithms
title: Find visible segments by sweeping
---

After two months of classes and two weeks of exams, the first quarter of my Master's is finally complete. I can say that 1/8th of my degree is complete. With a new quarter comes new courses. It's been just a week but there is one specific topic that I found quite intriguing: geometric algorithms, specifically sweep line algorithms.

Before getting deeper into the subject, what are geometric algorithms? Well, they are used to solve geometric problems! Now you may be thinking "why is he stating the obvious?". Since geometric problems usually deal with an enormous amount of diverse data, the computational complexity is very important. A difference between the running time of a bad algorithm compared to an efficient one can be in the order of days, or even months and years.

To illustrate this, I'm going to use a variant of the sweep line algorithm. Traditionally, the algorithm is sued to find intersections aong line segments in a plane but there are many variations. Let's picture the problem where you have a point _p_ and many line segments surrouding it. None of them interesect each other, being completely disjoint. A simple algorithm would be to go for each line segment and test it against all the others to see if it was the closest to _p_ at any point. However, the complexity of this is insane.

With that being said, there's a much better, yet simple, strategy that would solve our problem in logarithmic time. Please remember that we're assuming there's no intersections and all line segments are completely disjoint. The idea is to have a ray from _p_ that does a 360º roudtrip and checks for the closest line segment. However, how can we do this?

We're going to need two data structures: one for the events, which are the endpoints of the line segments (for starting and ending); and another one for the status, which will represent the current ray intersections at the current instant in time.

For the events, we will be using a priority queue _E_. The operations we need, enqueue and dequeue, can both be executed in constant time. For the status, we will be using a balanced binary search tree (BST) _S_ where the operations search, insert and delete can run in logarithmic time.

Let's look at the algorithm now:

1. Initialize _E_ with all the endpoints from all line segments. The endpoints are represented in polar coordinates _(angle, distance)_ and sorted by _distance_. This takes _O(n log n)_.
2. Initialize _S_ with the intersections for the first ray, when the angle of rotation is 0º. The BST will be sorted by distance. Takes _O(n)_.
3. For each event in _E_, remove the element from _S_ if it's an end endpoint or add the element to _S_ if it's a start endpoint. On both cases, check the leftmost leaf in the tree, which will represent the current visible line segment, since it is sorted by distance. Takes _O(n log n)_.

In total, the algorithm takes _O(n log n)_ time. Try to picture it. Does it make sense? It makes sense to me and I hope it's correct. 

What do you think about this kind of problems? There's many other variations of this, such as find intersections and find the smallest circles in a set of points.