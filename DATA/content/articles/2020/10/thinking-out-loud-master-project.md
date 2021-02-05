---
publishDate: "2020-10-05T09:00:00.000+02:00"
syndication:
- https://twitter.com/hacdias/status/1317859153185001474
tags:
- masters
- mondayletter
title: Thinking out loud about my future master's project
---

I'm now in my first year of my Master's degree in Computer Science and Engineering which means that next year I am supposed to start (and hopefully finish) my Master's project and thesis. As a consequence of that, I have been thinking a lot about it lately. I don't want to make a rushed choice that will make me work on something I don't enjoy for over half an year.

You may be thinking "of course he has some topics he would like to work on" and that is completely true. The issue is that I feel I don't have any highly specific topic to work on and they're all very superficial. On this post, I am going to write a bit about some topics I'm curious about or some questions that I would like to see answered. Hopefully, by writing, it will also help me to settle down some ideas.

The first aglomerate of ideas are related to the [decentralization] of the web. There is a million unanswered questions and topics to work on in this field. But first of all, why? The current model of the Innternet is mainly centralized. Even though the web wasn't created with those ideals, big companies ended up literally owning most of the online traffic.

Read this:

> For me the fact that when I send a text message to someone that is in my same room or in the house next door, this message has to travel from my device, to the backbone of the Internet, potentially be processed in a data center, and back to its destination really messes with my mind. - [Alfonso][alfonso]

Think about it for a second. If you're reading this, it's likely that you've come across this thought in the past, like me, and it is still always mesmarizing to think about it. When I tell people about it, they usually give a reply such as "really? did that happen so fast? why did that happen?". I recomend you to read Alfonso's view on [the future of the Internet][alfonso-view] because I share a lot of his ideas.

With all of this being said, to create a fully decentralized web, we have to address some issues. Issues that aren't yet solved or that are still being worked on. Or issues that have some solution right now, but it's still not the best it can be and should be improved. So let's get into some topics that look interesting to me by no particular order.

## Human readable naming

The Zooko's triangle says that we can only pick two criteria from these three: decentralized, human-friendly and secure. The big issue is that we want the 3 of them to be true at the same time. How ca we create a human readable naming system, better than DNS - which is by design decentralized, but has its own issues - that is efficient, collision free, human readable, decentralized and secure? There is a [document](https://github.com/protocol/ResNetLab/blob/master/OPEN_PROBLEMS/HUMAN_READABLE_NAMING.md) from ResNetLab that explains this problem in more detail.

## Identity in distributed systems

Another issue regarding the distributed systems is identity and identity management. On centralized services, we are usually connected to some service that stores information about us and we can log in/authenticate ourselves through some credentials that they can then compare with whatever they are storing.

On a decentralized system, how can we have our own identity and prove it? How can we own the data, securely, in a decentralized manner, to provide authentic and fidedign identity systems?

## Data Routing at Scale

Since we were talking about data, there is another big important point: how can data [move across the network](https://github.com/protocol/ResNetLab/blob/master/OPEN_PROBLEMS/ROUTING_AT_SCALE.md)? When a system is applied in large scale, it needs to work and for a decentralized system whise main goal is to move data across computers, it is essencial that the applied algorithms are efficient and reliable.

Not regarding _how_ we move data, but _from where_ we fetch it. The big issue here is how to do routing at large scale? Content-addressable networks, like IPFS, face this issue as the amount of addressable elements in the network rises by several orders of magnitude compared to the host-addressable Internet of today.

## Legal documents & blockchain

Still related with decentralization, another topic of interest for me is the application of blockchain applications to the legal framework of countries. What if legal documents and/or laws were stored in a blockchain, in a decentralized manner? Completely transparent.

Another interesting application is for high school and university diplomas. What if all diplomas got registered in a blockchain, providing authenticity and verifiability? It could be and work like a notary. This idea [isn't new](https://www.researchgate.net/publication/327483862_Blockchain_as_a_Notarization_Service_for_Data_Sharing_with_Personal_Data_Store)! However, I feel there's a lack of practical applications.
 
---

Well, that was all for some of the topics I'm interested in. But the list definitely doesn't end up there! Even though I can't recall any big topic now, there's certainnly a multitude of other topics I wouldn't mind to work with.

I still don't know quite well how Master's projects (and thesis) work. I know there's an [assignments] page for my department where some research groups have their own proposals. I'm also aware I can contact the research groups about some ideas so they can give me guidance.

For what I looked and could understand, the most closely related [research groups][groups] at TU/e to the topics I'm interested in are [ALGA][alga], the algorithms group, and [SAN], the system architecture and networking group.

Maybe I'm thinking too far ahead since this is just for next year. Am I?

[alga]: https://alga.win.tue.nl/
[uai]: https://uai.win.tue.nl/
[assignments]: https://assignments.win.tue.nl/
[san]: https://www.win.tue.nl/san/main/research/
[groups]: https://educationguide.tue.nl/programs/graduate-school/masters-programs/computer-science-and-engineering/graduation/cs-research-groups/
[alfonso]: https://adlrocha.substack.com/p/adlrocha-what-the-next-generation
[alfonso-view]: https://adlrocha.substack.com/p/adlrocha-my-vision-for-a-new-internet
[decentralization]: /tags/decentralization/