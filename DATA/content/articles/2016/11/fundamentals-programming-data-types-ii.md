---
publishDate: "2016-11-16T00:00:00.000Z"
title: 'The Fundamentals of Programming #6: Data Types II'
---

In our beginning articles, we mentioned that there are some
types of variables and constants and that is what we are going to talk today.
This is part of our Fundamentals of Programming Using C series.

<!--more-->

## Integer numbers

We will start by talking about integer numbers (e.g.: 4500, 5, -250). This type
of data is represented by the word int. With C, you can define the range of
numbers in which the content of a variable of int type can be.

**Initialization** is the process of giving the first value to a variable. This
first value doesn't need to be hard coded in the source: it can be the input of
the user for example.

In the following example, you can see how to declare a variable, it means, how
to reverse its space on RAM memory. This variable will be of type int and will
be named a. After that, it will be initialized with the value 20 and printed out
to the screen.

```c
#include <stdio.h>

int main() {      
    int a;
    a = 20;
    printf("I saved the number %d.", a);  
    return 0;      
}
```

## printf function and integer numbers

If we want to use printf function to print the content of a integer number variable, we have to use the placeholder %d. In the first parameter of the function, you should use that piece of code when you want to print an (integer) variable content and then use the variables, as other parameters. Look at this:

```c
#include <stdio.h>

int main() {
    int a, b;
    a = 20;
    b = 100;

    /* prints: The first number is: 20 */
    printf("The first number is: %d.\n", a);
    /* prints: The second number is: 100 */
    printf("The second number is: %d.\n", b);
    /* prints: The first and the second numbers are 20 and 100 */
    printf("The first and the second numbers are %d and %d.\n", a, b);

    return 0;
}
```

## short and long modifiers

You might want to checkout the code first.

The int type generally occupies between 2 and 4 bytes of memory. What if you want to use the variable for a small number? Can I save some resources? And what if I need a bigger number? In this situations you can use modifiers.

A modifier is a keyword that when put before an element can change one of its properties.

To change the storage capacity of a variable of the type int, in other words, the number of bytes that the variable occupies, we can use the modifiers short and long. These allow us to change the size of the variable to a bit smaller and a bit longer, respectively.

When you change the size of a variable of type int, you are also changing the range of values that the variable can hold. We have:

* 1 byte can store a number from -128 to 127
* 2 bytes can store a number from -32 768 to 32 767
* 4 bytes can store a number from -2 147 483 648 to 2 147 483 647
* 8 bytes can store a number from -9 223 372 036 854 775 808 to 9 223 372 036 854
775 807

To use these modifiers, you should proceed the following way:

```c
short int atoms = 20; // or long
```

## sizeof function

The number of bytes that is assigned to a variable when using this modifiers can depend on the computer where the code is being executed. This "problem" also depends on the language and there are some programming languages in which the number of bytes won't depend on the machine.

If you want to discover how many bytes a variable occupies in your memory RAM, you can just use the function sizeof this way:

```c
#include <stdio.h>

int main() {    
    printf("int : %d bytes\n", sizeof(int) );
    printf("short int: %d bytes\n", sizeof(short) );
    printf("long int: %d bytes\n", sizeof(long) );
    return 0;
}
```

In my computer, for example, a short int occupied 2 bytes, a long one 8 bytes and a "normal" one only 4 bytes.

## signed and unsigned modifiers

As you know, integer numbers can be either positive or negative. Sometimes, negative numbers may muddle or help depending on the case.

If we want to have control over the "positivity" or "negativity" of a number, we can use the modifiers signed e unsigned. If you want a variable to hold only positive numbers, you have to add it the unsigned modifier. If it can have both "types" of numbers, you may use the signed modifier.

Taking into account that the variables marked with unsigned cannot hold negative numbers, they will have a different range of positive numbers: it will support bigger numbers. If an ordinary int supports numbers between -32 768 and 32 767, the same variable with the unsigned modifier will support numbers from 0 to 65 535.