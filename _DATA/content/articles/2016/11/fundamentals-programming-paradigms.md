---
publishDate: "2016-11-12T00:00:00.000Z"
title: 'The Fundamentals of Programming #3: Paradigms'
---

Every single programming language has its own characteristics
that distinguish them from each other. One of those characteristics is the
paradigms they follow. That is something very important to understand how some
programming languages work. This is part of The Fundamentals of Programming
Using C series.

<!--more-->

**Programming paradigms** are styles or patterns that tell the "way" to program
in that language. There are many of them.

We will approach 6 programming paradigms, the best known and used. The languages
that adopt more than one paradigm are called **multi-paradigm** languages.

## Imperative paradigm

The first paradigm we are talking about is the imperative paradigm. This
paradigm consists in a **state** (which are variable) and **actions** (commands)
that are able to **change** the state.

This paradigm can be compared to the imperative tense in our language since it
is used to command the realization of action (for example, do something,
analyse, clean…).

Some examples of programming languages that follow this paradigm are: C, Java,
C#, Pascal.

## Procedural paradigm

The programming languages that follow the procedural paradigm, allow us to reuse
pieces of code without copying and pasting them in another place. This can be
done using functions and procedures (we will talk about this later).

The majority of the programming languages we use nowadays have adopted this
paradigm.

## Structured paradigm

In the programming languages that follow the structured paradigm, the source
code of an application can be reduced to only three structures: sequences,
decisions and iterations (repetitions).

## Sequence

In this type of structure, the tasks are executed in a linear way, it means, one
after the other. Look at this example:

```
WakeUp;
DressUp;
Breakfast;
Work;
```

In this sequence we can see some of the linear actions that are taken by almost
every human on Earth on their day to day. Here is the corresponding flowchart:

{{< figure class="invert" src="sequence.jpg" alt="Sequence flowchart" >}}

In most programming languages, the commands/actions finish with a semicolon so
they can be all written in one single line, like this:

```
WakeUp; DressUp; Breakfast; Work;
```

The usage of a semicolon is usually mandatory when there is more than one
instruction in one line. There are some programming languages that only require
us to put a semicolon in this cases (JavaScript, for example). But in the
majority of them we have to put a semicolon after finish each instruction (like
C, Java, C# and so on).

## Decision

In this kind of structure, there is a piece of code that is executed, or not,
depending on a logical test. We have got some examples for structures.

In the first example, we are going to write the condition "If I wake up, I will
work. Otherwise, I will not work" in pseudocode.

```
if "Wake Up" then        
  "Work"        
else      
  "Don't work"        
endif
```

As you can, it is easily comprehended. Note that Work will only be executed if
the individual Wake Up. Otherwise, the piece Don't work will be executed. Let's
see a pretty flowchart for this:

{{< figure class="invert" src="decision.jpg" alt="Simple decision flowchart" >}}

Now, let's see the second example. In this one, we will illustrate the condition
"I have an headache. If it's a light one, I will work. But if it is moderate, I
will take a pill and go to work. If it's severe, I will go to the doctor and
don't go to work".

```
case "headache"        
  when "light" then "work"        
  when "moderate" then "take pill"; "work"        
  when "severe" then "go to the doctor"; "don't work"
```

You probably thought that I would use "ifs" but I didn't. In almost every single
programming language, there is an alternative for the use of "ifs" when it all
depends on a variable content. Take a look at this flowchart:

{{< figure class="invert" src="decision-social.jpg" alt="Complex decision flowchart" >}}

This flowchart represents the code above, but in a different way. This flowchart
is exactly the same as the code below. And the code below is exactly the same as
the previous piece of code you've seen.

```
if "headache"          
  if "light" then          
    "work";          
  else if "moderate" then          
    "take a pill";          
    "work";          
    else if "severe" then          
    "go to the doctor";          
    "don't work";          
    endif          
endif
```

## Iteration

There is one more kind of structures: iteration, also known as repetition. In
this structures, a piece of code is repeated *n* number of times, usually
depending on a logical test.

Let's take a look at a pseudocode piece that represents the repetition "I will
not leave the house until I'm dressed":

```
do {        
  "not leave the house";        
} while ( "not dressed" )
```

The previous code can be read as: **do** "not leave the house" **while** "not
dressed". Basically, you can say, do *x *while *y*.

Let's take a look to one more repetition. This time it is "while I sleep, I
don't dress myself up":

```
while ( sleep )
  doNotDress();
```

It means, **while** something is happening, do other thing.

Now, let's see an example to "brush the teeth 20 times":

```
for ( i = 0; i++; i < 20)
  brushMyTeeth();
```

The last example is "for each teeth, I clean it very well":

```
for each teeth in mouth
  cleanVeryWell();
```

It means, for each item of a set, do something.

We will talk more about this kind of structures afterwards. They're essential.

## Declarative paradigm

The declarative paradigm contrasts with the imperative one because it is able to
express the logic without telling how they work. In other words, using this
paradigm, the programmer tells the computer **what** to do but not **how** to
do.

One well known language that follows this paradigm is Prolog, which is somewhat
used in Artificial Intelligence field.

## Functional paradigm

At a first glance, you could think that it's called functional paradigm because
it uses functions (that's what I thought. Oops), but it isn't. It's called like
this because it uses mathematical expressions and functions and avoid states.
These languages are heavily used on Math field.

Some programming languages that follow this paradigm are, for example, Matlab,
Wolfram Language, B, etc.

## Object oriented paradigm

With the Object oriented paradigm, we can create **objects** based on
**classes**. Those objects are instances of those classes and they have the same
attributes and functions that the class have.

This paradigm is really extensive and there are a lot of programming languages
nowadays that support this paradigm: Java, C++, C#, PHP, etc.

*****

Note that I have only talked about 6 programming paradigms, but there are a lot
more. These are the most comprehensive ones. There are paradigms based on
paradigms, there are paradigms that contrast with other paradigms, etc.