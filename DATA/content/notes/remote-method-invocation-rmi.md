---
title: Remote Method Invocation (RMI)
---

The **Remote Method Invocation** (RMI) is a Java API that extends the concepts provided by [Remote Procedure Call](/notes/remote-procedure-call/), but using an [object-oriented](Object-oriented%20Programming) approach.

## Similarities with RPC

- Usage of interfaces to define the methods that can be remotely invoked.
- There is a protocol of remote invoking that offers the same [failure semantics](/notes/remote-procedure-call/#failure-semantics) of RPC.
- The transparency level is similar in the sense that the developer is only exposed to really few amount of distribution details.

## Differences with RPC

- You can use OOP with RMI and all of its concepts: objects, classes, inheritance, polymorphism, etc.
- Not required to use an IDL since the OOP languages are, by nature, strongly typed and already have the interfaces concept built-in.
- Passage by reference is now allowed since each object, local and remote, has an unique identifier.
- In each server, there can be more than one object and interfaces, while in RPC 1 server can only provide 1 distinct interface.

## Model

- Local objects: can only receive local method invocations.
- Remote objects: can receive both local and remote invocations.
- Since each object has a unique ID, its methods can be called in the remote server, by sending the ID, alongside with the request.
- Garbage Collection must be remotely supported.
- The remote calls can throw exceptions.