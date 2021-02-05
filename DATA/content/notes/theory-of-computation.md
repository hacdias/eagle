---
title: Theory of Computation
---

## Languages

### Syntax

A language's syntax defines what is valid on that language. For example, is a language syntax defines that all variables must be declared as `name := value`, you can't write it as `var name = value;` instead.

The syntax of a language is specified using a [grammar](#grammars).

### Semantics

Semantics attributes a meaning to valid sentences, or forms.

## Grammars

A grammar is a set of rules composed by:

- Non-terminal symbols
- Terminal symbols
- Rules

## BNF Language

BNS stands for Backus–Naur form and it is a notation technique to represent context-free grammars.

- `<..>` - Non-terminal Symbols
- `...` - Terminal Symbols
- `::=` - Defined as...
- `|` - Or
- `*` - Zero or more
- `+` - One or more

Example:

```
<S> :: = <A>a
<A> :: = a<B>
<B> :: = <A>a|b
```

Terminal symbols: `a, b`
Non-terminal symbols: `<S>, <A>, <B>`