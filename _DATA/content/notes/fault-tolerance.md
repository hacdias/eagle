---
math: true
title: Fault Tolerance
---

There are different type of [faults](/notes/fault-error-failure/) and we need metrics to work on measure how well software can tolerate them. These metrics allow to quantify and compare different systems.

## Reliability

To measure the reliability, we measure the medium time since the initial instant to the fault.

- **Mean Time To Failure (MTTF)** - non-reparable systems.
- **Mean Time Between Failures (MTBF)** - this needs to ⬆️
- **Mean Time To Repair (MTTR)** - this needs to ⬇️

## Availability

The availability $a$ is given by the composition of the previous concepts:

$$a =\frac{MTBF}{MTBF+MTTR}$$

## Fundamental Models

It is a good practice to define a fault model beforehand. This is one of the three fundamental models:

- Interaction model
- Fault tolerance model
- Security model

### Interaction Model

- Communication channel
  - Latency
  - Bandwidth
  - Does it support unordered messages?
  - Can the messages be duplicated?
  - Time sync
- **Synchronous**
  - Each message arrives at the destination in a given time limit
  - The time to execute each task is between known limits
  - The deviation of the local clock from the absolute time has a known limit
- Otherwise it is **Asynchronous**

**How to detect faults?**

- **Synchronous** system:
  - Assumptions:
    - Maximum latency
    - Maximum processing time
  - If the time limits are exceeded, then there is a fault
- **Asynchronous** system:
  - Impossible to limit the time latency and response time
  - Impossible to detect remote faults because they can be confused with latency increases.

## Fault Tolerance Model

- Defines which are the expected faults
- Defines which can be or not tolerated

In a [**distributed** system](/notes/distributed-systems/), the fault tolerance model is much more complex than their centralized counterpart because there are many points of the system that can fail:

- Communication faults
- Node failures (processing, system, servers, clients, storage, etc)

For this kind of models, we need to take into account two main different types of faults:

- **Fail-silent Fault**: when the component stops replying to any external stimulation.
  - Can be detectable: fail-stop
  - Or non-detectable: crash
  - **Processes**: stop responding.
  - **Communication channel**
    - **Send-omission**: lost between the sending process and the output buffer.
    - **Channel-omission**: lost between any buffer in the way.
    - **Receive-omission**: lost after arriving at the input buffer.
- **Byzantine Fault**: when any behavior is possible, e.g., return an incorrect output.
  - Worst case possible
  - Useful to represent software errors 
  - **Processes**
    - Does not reply to stimuli
    - Replies when there are no stimuli
    - Wrong replies to stimuli
  - **Communication channel**
    - Corrupted content
    - Delivers nonexisting message
    - Delivers duplicated messages
    - Does not deliver messages
    - Rare, but usually detectable
- **Dense Fault**: accumulation of so many tolerable faults that it becomes unbearable.

Usually, the models only take into account fail-silent faults unless proven otherwise. Also, dense and byzantine faults are not usually considered.

## Policies

- **Redundancy**
  - **Physical**: duplication of data or components.
  - **Temporal**: repetition of actions.
  - **Information**: algorithms that calculate the correct state based on the current state (e.g., parity bit for error recovery).
- **Recovery**
  - Replaces the bad state by a correct state, reverting some actions.
  - This implies that:
    - We can detect the error
    - We can calculate a previous (or posterior) state
  - While the system's recovering, the system stays **unavailable**, affecting its availability.
- **Compensation**
  - Computes the correct state from redundant components even if the internal state has something wrong so it is not needed to detect wrong states.
  - If there's enough redundancy, the recovery time is really quick, increasing availability.