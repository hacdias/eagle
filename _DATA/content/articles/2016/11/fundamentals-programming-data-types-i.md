---
publishDate: "2016-11-15T00:00:00.000Z"
title: 'The Fundamentals of Programming #5: Data Types I'
---

In our beginning articles, we mentioned that there are some
types of variables and constants and that is what we are going to talk today.
This is part of our Fundamentals of Programming Using C series.

<!--more-->

**Data types** consist in a variety of values and operations that a
variable/constant of that kind can support. They're needed so the compiler (or
interpreter) knows which conversations are needed to get the data from memory.

We can divide the data types in two types: primitive types and composite types.

**Primitive data types**, natives or basics, are those which are offered by a
programming language as basic construction blocks.

**Composite data types** are those that can be built using primitive data types.

## Variables

We have already mentioned the existence of variables and constants, which are
used to hold values, previously. And now, we I am going to explain you how to
declare variables in the language we are going to use from now on: C.

```c
type name;
```

Where:

* type is the type of the data the variable will hold; and
* name the name of the variable.

Imagine that you would want to create a variable named age which can hold an
integer number. You could do it this way:

```c
int age;
```

## Constants

In C, like in many other languages, there are constants. I remind you that
constants are those that do not let you to change its value while the program is
running. There are two ways to create constants in C: the declared constants and
defined constants.

## Defined constants with #define

We call defined constants to those which are declared in the header of a file.
These are interpreted by the preprocessor which will replace the constant by its
value in the whole file before being compiled.

```c
#define identifier value
```

Where:

* identifier is the name of the constant which is, conventionally, written in
capital letters and underscore to separate the words; and
* value which is the constant's value.

Imagine now that you want to have a constant for the value of Pi and then you
want to determine the perimeter of a circle. You could do it this way:

```c
#include <stdio.h>
#define PI 3.14159

int main (){
    double r = 5.0;              
    double circle;

    circle = 2 * PI * r;      
    printf("%f\n", circle);
    return 0;
}
```

On line 2, you are indicating the preprocessor to replace all the occurrences of "PI" by "3.14159", literally.  You can use the library `math.h` which includes the constant `M_PI` that holds the value of Pi.

## Declared Constants

The Declared Constants, as opposed to the Defined Constants, are declared in the code. The declaration of this type of constants is very similar to the declaration of a variable. We just need to write the const keyword before. Like this:

```c
const type name = value;
```

Where:

* type is the data type;
* name is the constant's name; and
* value the value of the constant.

I you try to change the value of a constant during the run time, you will get an error. Take a look at this code:

```c
#include <stdio.h>
#include <math.h>

int main() {
    const double goldenRatio = (1 + sqrt(5)) / 2;

    goldenRatio = 9; // error

    double zero = (goldenRatio * goldenRatio) - goldenRatio - 1;
    printf("%f", zero);
    return 0;
}
```

There are some advantages of both types of constants. The biggest advantage of declared constants is that they can be declared locally and globally.

A local variable/constant is a variable/constant that is limited to a function and that only can be used within that function.