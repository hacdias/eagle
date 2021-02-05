---
tags:
- books
title: The Practice of Programming
---

- **isbn**: 0-201-61586-X
- **initial read**: 2018-05-25
- **quick reread**: 2020-06-16

A quick note about the book: a lot of the things are C-specific and do not apply to other languages such as tips with macros, etc. This is basically a "how to be a good programmer" book.

## Chapter 1: Style

- Names are important
- Be clear and consistent
- Follow conventions
- "Parenthesize to resolve ambiguity"
- "Break up complex expressions"
- "Be clear"
- Do not write bad code
- Write readable code and comments

## Chapter 2: Algorithms and Data Structures

- [Search Algorithms](/notes/search-algorithms/)
- [Sorting Algorithms](/notes/sorting-algorithms/)
- [Data Structures](/notes/data-structures/)
- [Asymptotic Notation](/notes/asymptotic-notation/)

## Chapter 3: Design and Implementation

## Chapter 4: Interfaces

- Hide implementation details
- Choose a small orthogonal set of primitives
- Don't reach behind the user's back
- Do the same thing the same way everywhere (consistency!)
- Free a resource in the same layer that allocated it

## Chapter 5: Debugging

- Look for familiar patterns
- Don't make the same mistake twice
- Debug it now, not later
- Get a stack trace
- Read before typing
- Explain your code to someone else (or a rubber duck!)
- Make the bug reproducible
- Divide and conquer
- Study the numerology of failures
- Display output to localize your search
- Write self-checking code
- Write a log file
- Draw a picture
- Keep records
- Use correct and adequate debug tools

> If you think that you have found a bug in someone else's program, the first step is to make absolutely sure it is a genuine bug, so you don't waste the author's time and lose your own credibility.
> ...
> Finally, put yourself in the shoes of the person who receives your report. You want to provide the owner with as good a test case as you can manage. It's not very helpful if the bug can be demonstrated only with large inputs, or an elaborate environment, or multiple supporting files. Strip the test down to a minimal and self-contained case. Include other information that could possibly be relevant, like the version of the program itself, and of the compiler, operating system and hardware.

## Chapter 6: Testing

> It is important to test your own code: don't assume that some testing organization or user will find things for you. But it's easy to delude yourself about how carefully you are testing, so try to ignore the code and think of the hard cases, not the easy ones. To quote Don Knuth describing how he creates tests for the TEX formatter, "I get into the meanest, nastiest frame of mind that I can manage, and I write the nastiest [testing] code I can think of; then I turn around and embed that in even nastier constructions that are almost obscene."

- Test code at its boundaries
- Test pre- and post- conditions
- Use assertions
- Program defensively
- Check error returns
- Test incrementally
- Test simple parts first
- Know what output to expect
- Verify conservation properties
- Compare independent implementations
- Measure test coverage
- Automate regression testing
- Create self-contained tests

## Chapter 7: Performance

- Buffer data
- Use caches
- Save space
- Estimate
- Choose the right algorithms

## Chapter 8: Portability

- Standardization is important
- Internationalize software
- Try not to use features not available on all OSes
- Isolate dependencies
- Version compatibilities

## Chapter 9: Notation