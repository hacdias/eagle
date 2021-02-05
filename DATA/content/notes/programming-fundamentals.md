---
title: Programming Fundamentals
---

An algorithm is a finite sequence of unambiguous and well-defined instructions. Each instruction can be mechanically executed in a finite period of time with a limited amount of effort in order to reach a goal.

1. **Strict**: each instruction specifies exactly what to do. There can't be ambiguity.
2. **Efficient**: each instruction must be basic enough to be executed in a limited period of time with limited effort.
3. **Must end**: an algorithm must lead to a state in which the goal was reached and there are no more instructions to be executed.

## Program

A program is a sequence of instructions that are executed by a computer, defining its behavior. They are algorithms written in programming languages.

## Computational Process

Something that exists inside a computer during a program's execution and whose evolution is strictly defined by the program itself.

## Syntax Errors and Semantical Errors

[**Syntax errors**](/notes/theory-of-computation/#syntax) are caused by language-specific violations, i.e., each language specifies its own syntax and that must be followed in order for the program to run.

[**Semantic errors**](/notes/theory-of-computation/#semantics) are harder to debug. They are syntactically correct, but the meaning and the logic behind is wrong.

## Program Development Phases

1. **Problem Analysis**: the programmer alongside with the client, study the problem in order to find know exactly what needs to be done. Clearly specify the problem.
2. **Solution Development**: determine on how the problem is going to be solved. Develop an abstract algorithm of what's the solution is going to look like.
3. **Solution Coding**: translating the pseudo-algorithm to a programming language, implementing the solution.
4. **Testing**: testing must be done in order to "guarantee" that the program will behave correctly for known use cases. It's still impossible to test for every scenario because they're manually defined by us and we can't think about all the possible scenarios.
5. **Maintenance**: after the program starts being used in production it will need maintenance in order to keep up to date and working.

## Abstraction

### Procedural Abstraction

Procedural abstraction is knowing what a function (procedure) does but now knowing how it is implemented. This concept is mainly used when controlling the **complexity**. First, we define the procedures we want and what they need to do, and only afterwards they get implemented.

### Data Abstraction

Similarly with procedural abstraction, this concept consists on considering some properties of a [data structure](/notes/data-structures/), while ignoring how it is represented.

By defining a set of basic operations that can manipulate a certain abstract data type, the rest of the program can use those operations to manipulate the object without knowing how they're being internally handled. By messing with the internal representation of data structures, we would be **violating abstraction barriers**.

## Abstract Data Types (ATD)

An abstract data type defines the operations from the point of view of the user, i.e., it defines the behavior of the operations. They define the **interface**.

### Defining an ATD

Defining an ATD before implementing the data type is a method that is used in order to define the basic operations and their behavior.

1. Identify the basic operations:
   - **Constructors**: `create_real: int x int --> real`
   - **Selectors**: `numerator: real --> int`
   - **Recognizers**: `is_real: any --> boolean`
   - **Comparators**: `equal_reals: real x real --> boolean`
   - **Transformers**  `write_real: real -->`
2. Identify the relations that the basic operations must agree on.
3. Define the internal representation.
4. Implement!

## Parameters

- **Parameters** specify the input data for a certain routine.
- **Arguments** is the actual input that is used when invocating a routine.

Arguments can either be passed by value or reference.

- When passing by **value**, the value is copied to the routine and any changes that are made inside of it are not reflected outside of the routine. It's an unidirectional operation from the point of view of the routine's caller.
- When passing by **reference**, we are actually passing the memory location of the value. Hence, any changes made inside the routine will be reflected outside of it.

When a function is called, a local environment is created, in which the arguments and the parameters are associated. This environment is destroyed when the routine ends.

## High-order function

A high-order function is a function that satisfies at least one the following conditions:

- Takes one or more functions as arguments;
- Returns a function as its result.

## Recursion and Iteration

### Recursive Process

A recursive process is characterized by having an phase of expansion, followed by a phase of contraction.

Example of a linear recursion process:

```
rec(2, 3)
| 6 + rec(2, 2)
| | 4 + rec(2, 1)
| | | 2 + rec(2, 0)
| | | | 0
| | | 2
| | 6
| 12
12
```

### Iterative Process

It is characterized by a certain amount of variables, called state variables, alongside with a specific rule to update them. The state variables provide a complete description of the state of the computation in each moment.

## Programming Paradigms

There are some different programming paradigms that define the overall look and behavior of a certain language:

- [Imperative Programming](/notes/imperative-programming/)
- [Functional Programming](/notes/functional-programming/)
- [Object-oriented Programming](/notes/object-oriented-programming/)