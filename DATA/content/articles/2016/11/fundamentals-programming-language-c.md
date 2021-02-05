---
publishDate: "2016-11-14T00:00:00.000Z"
title: 'The Fundamentals of Programming #4: Language C'
---

The years between 1969 and 1973 were very exciting for the
AT&T Bell Labs because it was when the programming language C was mostly
developed. This is part of The Fundamentals of Programming Using
C series.

<!--more-->

The main developer of this language was **Dennis Ritchie** who described the
year of 1972 as the most productive and creative one.

The language developed by Ritchie is called **C** because this language was
based on other programming language, called **B**. B and C have some aspects in
common.

Initially, the language C was mostly developed with the objective to improve
Unix, which was already written in **Assembly** — another programming language.

The latest version of C is C11 and was released in December 2011. This language
is one of the biggest influences in the programming world. C influenced, among
others, AWK, BitC, C++, C#, C Shell, D, Euphoria, Go, Java, JavaScript, Limbo,
Logic Basic, Objective-C, Perl and PHP. **This doesn't mean that this languages
weren't influenced by others**.

We will start by talking about C because it can be somehow called the "mother"
language. It's a low level programming language, so it can have a lot of contact
with the hardware.

## C language characteristics

C is a structured, imperative and procedural programming language. Do you
remember the [paradigms](/articles/2016/11/fundamentals-programming-paradigms/)?
Other characteristics of this language are being standardized by ISO and being
of general purpose.

## It's compiled

C is a compiled programming language. It means that a C program, before being
executed, must pass through a process, in which the source code is converted to
the machine code. The tool that does this is called **compiler**.

If you learn C, you will be well prepared to learn a lot of other languages that
are based on it. The syntax is very similar across many languages but the most
important it's the logic.

## Development environment

To start programming, you need to set a development environment with the needed
tools.

## Compiler

A **compiler** is a tool that translates your source code into the machine code
through a process called compilation.

We will use the compiler GCC, which stands for GNU Compiler Collection. This
compiler is very easy to use and to install.

**Debian and derivatives**

The installation of GCC on Debian distribution and derivatives, like Ubuntu, is
very easy. You just need to open your terminal and run the following commands:

```console
$ sudo apt-get update && apt-get upgrade
$ sudo apt-get install build-essential
```

After that, you should check if it was correctly installed. To check that, you
need to run this command:

```console
$ gcc -v
```

This command will return the current installed version of GCC if everything went
well.

**Other Linux distributions**

To install GCC on other Linux distributions, you will need to follow the
instructions on their [official page.](https://gcc.gnu.org/)

**Windows**

On Windows, you can install GCC with projects like MinGW or Cygwin. We recommend
the [first](http://www.mingw.org/download/installer).

**OS X**

On Apple's operating system, GCC comes with XCode, a multi-language IDE built by
Apple. It can be installed via terminal with the following command:

```console
xcode-select --install
```

You should check the GCC version to see if everything was correctly installed
(see the command above).

An **IDE** is an Integrated Development Environment. That kind of programs
include the needed tools to help during the development process.

**Text editor**

Beyond the compiler, you will also need a text editor. Even the notepad works,
but we recommend you to use one with syntax highlight.

There are a lot of text editors you can use. We leave you some recommendations
here:

* **Windows ->** Notepad++, Atom, Sublime Text;
* **Linux** -> Gedit, Atom, Sublime Text (alguma distribuições), Vim;
* **OS X** -> TextWrangler, Sublime Text, Atom.

## "Hello, World!"

How would the programming world be without the famous "Hello, World!"? It's
tradition to be a programmers' first program showing the line "Hello, World!" on
the screen.

Create a file, wherever you want, with the name HelloWorld.c. Note that you must
use the extension .c so the text editor can know in which language the file is
written.

Open the file you just created and copy and paste the following code into it:

```c
#include <stdio.h>   
#include <stdlib.h>

int main() {
    printf("Hello World!\n");      
    return 0;
}
```

This piece of code will print in your screen the message "Hello World!". To run your program, you will need to compile it first. So, you have to open your Terminal/Console and run the following command:

```console
$ gcc HelloWorld.c -o HelloWorld
```

Where HelloWorld.c is the input file and HelloWorld the output file. The extension of the built file will only depend on your operating system.

In my case, it is Windows, so the created file is HelloWorld. Now, to execute your program, you just have to type this on your command line:

```console
$ HelloWorld
```

And the message will be printed! Try with different sentences if you want.

## What's that #include?

The first line of the code above is not C code, but an indication to the compiler.

C is commonly used on places where we need to get high speeds, like on Linux's kernel. As C is an high speed language, it isn't prepared to do a lot of tasks "out of the box" so we need to include some files to get access to more functions.

To include more "sets" of functions we have to use the directive #include that tells the compiler to include header files. In this case, we are importing stdio.h which stands for standard input/output.

## Function main

Every single C program must contain a main function which is going to be automatically executed. This function is the start point of a program. We will talk more about this later.

## Function printf

This command/function is imported by the file stdio.h. If we don't include that file, a error will be generated. This function stands for print formatted.

This function accepts more than one parameter which are values that are going to be processed by the function. We will only talk about the first parameter now.

The first parameter is a string, it means, is a sequence of characters. This sequence of characters must be put between two quatation marks.

In this case, we wrote Hello World!\n which means that what is going to be printed is the message Hello World! and a new line \n.

The backslash (\\) is used to insert special characters.

| Symbol | Meaning |
| --------|-------------|
| \\%      | Percentage   |
| \t      | Tab         |
| \r      | Carriage Return |
| \a e \7 | Some sounds |

## Directive return

As you should have noticed, the function main returns an integer number. In this case, it returns 0, which is represents no error.

In this moment I will not tell you more about this directive, but it will be discussed later.

## Comments

We can comment our code to make it more easier to read for other programs or just to give more information about how it works. Sometimes it isn't clear enough and even we can forget how our own code works. The comments are ignored by the compiler.

There are two types of comments in C: single line comments and multi-line comments. The comments that begin with // go till the end of a line. And the comments that begin with /* and finish with */ make everything between them a comment.

Comments shouldn't be used to obvious things like "here we sum two variables". They should only be used when necessary or to add additional information.

```c
#include <stdio.h>

int main() {        
    int n = 1; // This is a single line comment

    /*  
    All of this is a comment

    Lorem ipsum dolor sit amet, consectetur adipiscing elit. 
    Aenean tempus, lectus at elementum mollis, velit magna mollis urna, 
    quis aliquam ligula est vel sapien. Mauris lacinia, turpis id fringilla 
    dapibus, eros nulla condimentum tortor, ut consectetur velit sem eu mi. 
    Mauris consequat cursus efficitur.    
    */   
}
```

The most frequent type of comment used in C is the one that begins with /*, independently of its size.