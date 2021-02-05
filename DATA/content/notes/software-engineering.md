---
title: Software Engineering
---

Software Engineering is the discipline that is concerned with:

- All aspects of software production;
- Creating cost-effective solutions;
- Creating solutions for customer's problems that are:
  - Maintainable,
  - Reliable; and
  - Efficient.

## As a learning process

- The client knows more about its own business after the development
-	Knowledge about the problem and the solution is created as the software artifact is built
-	The team knows more about the business after the development

## Complexity

Software Engineering complexity is basically size complexity, i.e., the more the size of the project grows, the harder it is to manage the project. Even if the project's code is not complex by itself, i.e., the algorithms are not complex, if there's thousands of lines of code, the management of such an amount of code starts to be a heavy burden.

## Lehman's categories of software systems

- **S-system:** written according to an exact specification of what that program can do. For example, a math library.
- **P-system:** written to implement a model of the problem, not the problem itself. To be executed by a computer, there needs to be precise semantics. However, the problem cannot be fully specified. The example given by Lehman is a chess game, where we can't specify completely how to win the game.
- **E-system:** written to implement a model of the problem, performing some real-world activity. The way it behaves is intrinsically connected to the environment in which it runs. For example, an automated stock trading software.

## Project Management

Software project management addresses the size complexity of software development. Usually, this is divided into three different scopes: team, software and functionality.

- How much is the project going to cost?
- How many developers will be needed?
- How long will it take?

### Team

Communication is needed between the members of a team and the quality of the communication itself depends on the use of a shared common language in order to reduce the risk of noise. For example, defining [code conventions](https://en.wikipedia.org/wiki/Coding_conventions) can be considered a step to enhance communication.

A **common language**[^cl] improves the quality of communication among the team members by standardizing the information that flows through the communication channels. Even though some expressiveness of language might be lost, it's more advantageous to have a strict common language where both the client and the team can communicate with less risks.

**Team organization**[^to] encompasses a set of rules to define how to group people in an organization, by taking into account their competences, skills and the organization's needs. In addition, it defines the communication channels, their number and the type and amount of flowing information.

## Git

See [Git](/notes/git/).

## Verification and Validation

Verification and validation should test for [faults, errors and failures](/notes/fault-error-failure/).

- **Verification** is done according to the specifications.
- **Validation** is done according to the client's needs.

## Testing

Testing requires:

- Programmers
- Clients
- Quality Assurance (QA) Engineers

Testing strategies:

- Testing the specification.
- **Black-box testing**: tests the functionality. Gives the input and check if the input is correct. Does not know anything about the inner working of the application.
- **White-box testing**: tests internal structures or working of an application.

[^cl]: https://antonioritosilva.org/software-engineering-companion/project-management/common-language/
[^to]: https://antonioritosilva.org/software-engineering-companion/project-management/team-organization/