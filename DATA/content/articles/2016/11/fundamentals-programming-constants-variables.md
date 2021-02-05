---
publishDate: "2016-11-11T00:00:00.000Z"
title: 'The Fundamentals of Programming #2: Constants and Variables'
---

Today we are going to discuss about constants and variables,
which are the "things" that can hold data for us. This is part of The
Fundamentals of Programming Using C series.

<!--more-->

Look at this code:

```
variable character letter <- "T"
```

In that piece of code, we declare a variable of the type **character**, whose
name is letter, with the value "T".

The **declaration of variables**, or constants, is the process in which the
compiler is "warned" about its existence. In this way, every time the name of
the variable/constant is mentioned in the code, the previously stated value can
be used.

When we declare the variable letter, we are reserving an address and space in
RAM memory to store that value. In this case, the value was assigned at the same
time of its declaration using the operator <-.

Since the moment we declare a variable or a constant, its reserved space in RAM
memory will be accessible using its name. Whenever the word letter is mentioned
in the code, the value of the variable will be returned.

## Constants

Let's talk first about constants.

**Constants**, as the name suggests, allow us to store unchangeable values, in
other words, the value of a constant can't be changed after its declaration.

Let's check an example:

```
constant string BLOG <- "Bits N Me"    
BLOG <- "Bits N Me" // This would produce an error
```

In the first line, we declare the constant BLOG and assign to it the value "Bits
N Me". Then, in the second line, we try to change it's value. But since it is a
constant, an error will be produced because its content cannot be changed during
the run time of the program.

## Nomenclature

Conventionally, the name of a constant is all written with capital letters so we
can easily distinguish constants from variables inside the source code. Using
non capital letters will not produce any error. This is only a convention.

Although, following this rule, it will be easier for you, and for any other
developer that sees your code, to distinguish between a variable and a constant
without looking for its declaration.

## Variables

On the other hand, we have variables.

**Variables**, as opposite to constants, allow us to store values that can be
changed during the execution of the code. They're heavily used to store states
and they're fundamental in programming.

Let's see an example:

```
variable string subject <- "Constants"      
subject <- "Variables"
```

In the first line we declare a variable, whose name is subject, with the value
"Constants". Then, we change its value to "Variables". No error will be produced
because variables allow us to change its value along the execution of the
program.

## Nomenclature

There are two main nomenclatures when we talk about variables names and it
depends on the language. There are languages that follow the variation
*loweCamelCase* of the pattern *CamelCase*. But there are other languages that
prefer the name of the variables to be *separated_by_lines*.

## Naming rules

There are some must-follow rules when we talk about the name of variables and
constants. If we don't follow this rules, an error will be produced.

The name of variables and constants must:

* not begin with numbers (e.g.: 9food is not allowed, but food9 is);
* not be equal to a reserved word (e.g.: if is not allowed, but maria is); and
* not contain special characters (there are some exceptions depending on the
language).

**Reserved words** are those words that are inserted into the language own
syntax. If the word "for" is part of a language's syntax, it can't be used as a
variable (or constant) name. If you try to do it, you will receive an error.

Variables and constants can be of different data types. There are some basic
data types that are available in almost every language. But it can vary from
language to language. Later, we will talk about the available data types in C.