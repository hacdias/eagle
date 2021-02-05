---
publishDate: "2020-10-19T09:00:00.000+02:00"
syndication:
- https://twitter.com/hacdias/status/1318089432113815553
tags:
- testing
- decentralization
- testground
- mondayletter
title: Testing peer-to-peer systems with Testground
---

"Test, test, test" is a well known sentence right now due to the current circumstances. However, it does not apply only to Covid-19 or other diseases. Testing is one word that all of us have dreamt - or had nightmares - about in the past. It is also one crucial part and step in the Software Engineering and Development process, as it allows to verify and ensure that a certain system behaves as expected. At least, for known cases!

As such, all kinds of systems should be able to be easily tested, regardless of what they do. Nevertheless, both you and I know that that is not the case. There are certain systems that are much easy to test than others, in many orders of magnitude. For example, a library that converts between different times formats is much easier to test than an app to upload and download files.

Furthermore, any system that involves peer-to-peer communication or any other kind of distribution gets much harder to test! At Protocol Labs, we wanted to know and measure how changes to the IPFS and libp2p codebase would affect the performance of the network, but we couldn't find any reliable platform to help us with that. And then... *Testground was born!* 🚀

Quoting the page, "[Testground](https://docs.testground.ai/) is a platform for testing, benchmarking and simulating distributed and peer-to-peer systems at scale. It is designed to be multi-lingual and runtime-agnostic, scaling gracefully from 2 to 10k instances, when needed".

In this post, I will guide you on how to create a simple test for two instances, where they should ping pong with each other. This is also one of the [example](https://github.com/testground/testground/blob/master/plans/network/pingpong.go) tests we use to exercise the platform during testing. Even a platform for testing needs to be tested, right? I will go over how to set-up Testground, explain some of the base functionalities and write the test with you!

## First, the features! 🌟

First and foremost, Testground is _packed_ with features. Thus, I won't go over all of them, but I will mention the most important - or relevant in my opinion - for this post.

1. **Tests are written as if they were unit tests**. There's no need for puppeteering or to package your entire system as a separate daemon.
2. The **runtime environment for tests** is simple, normalized and formal. There is a contract between your test plan and the Testground daemon. On one hand, Testground injects a set of environment variables and the test plan is supposes to emit events to the stdout, and assets and outputs to the outputs directory. And this is what allows tests to be written in any kind of programming language.
3. We support a **coordination API** powered by [Redis](https://redis.io/). Since the platform is intended to test distributed systems with many, many instances, it is required to have a coordination API that allows us to choreograph and coordinates actions between the nodes.
4. **Network traffic shaping** is very easy to set up. You can change latency, jitter, duplication, packet corruption and a few other settings to simulate many network conditions, similar to the real ones.
5. **Outputs** from runs are straightforward to collect, from logs, to any other artifacts produced by the test.

These are just some key features, you can read more about it in the [documentation](https://docs.testground.ai/). However, I would recommend checking out the [GitHub page](https://github.com/testground/testground). As an ever evolving project, it is easy that something might get forgotten in the documentation.

## Set everything up! 🛠

Testground is built in Go and currently we don't distribute binaries, so you will need to build it from source. Besides that, we will also need Docker to run the containerized tests. So... for you, please:

1. Install Docker: [documentation](https://www.docker.com/).
2. Install Go: [documentation](https://golang.org/dl/).

Now, let's install Testground! So, first of all, open your favorite new shiny terminal app and `cd` to some directory you want to. Then, follow the following steps:

```bash
$ git clone https://github.com/testground/testground.git
$ cd testground
$ make install # Builds Testground binary and the required Docker images
```

As of now, you should have Testground installed. To start it, please run `testground daemon`. The daemon is the orchestrator. Testground is built with a client-server view. The goal is to allow users to have a remote daemon on a server so they can send tests there to avoid running them on their machines. But, for this guide, we can run everything locally. 

## Now, the test... 📄

Now, with the Testground daemon already running, it's time to finally build our ping pong test. Are you ready? Ready, set, go! 🏃‍♀️ Start by creating a directory called `my-plan`. Inside it, we will need to create a file called `manifest.toml`.

I will not enter the [details](https://docs.testground.ai/writing-test-plans/test-plan-manifest) about it right now, but let's say this file is here to explain to Testground what the test is about, as well as defining which runners and builders can be used.

You can simply copy and paste the following content:

```toml
name = "my-plan"

[defaults]
builder = "docker:go"
runner = "local:docker"

[builders."docker:go"]
enabled = true

[runners."local:docker"]
enabled = true

[[testcases]]
name = "ping-pong"
instances = { min = 2, max = 2, default = 2 }
```

As you can see, it's pretty simple and raw: there's a test plan, called "my-plan". By default, it builds with `docker:go` and runs with `local:docker`. Inside this test plan, there's one test, called `ping-pong` which needs to have, exactly, two instances to work.

After storing the manifest, please import the plan into Testground:

```bash
$ testground plan import --from my-plan
```

As of now, Testground knows about your plan. If you're curious, you can run the command `testground describe --plan my-plan` to see a quick overview of the test plan you're building right now.

Let's start building! For this example, I will pick Go for the sake of easiness. Even though you can use Testground with any language, we have a full working SDK built in [Go](https://github.com/testground/sdk-go) (and another coming in [JavaScript](https://github.com/testground/sdk-js)) that are usefull to abtract some things from the test. If you decide to go for another language, you will just need to implement some functions to contact with the orchestrator, the sidecar!

Let's start by creating a `main.go` file and defining the main function:

```go
package main

import (
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func main() {
	run.Invoke(pingpong)
}

func pingpong(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
  // Our test will be here!

  return nil
}
```

As you can see, it's pretty straightforward: it's just a normal Go program, with a main function. Inside it, we call an invoker that will call our function `pingpong`. This function will, in turn, contain all the code required for our test!

From now on, all the pieces of code I will write down, should go inside the test function (except imports)!

Let's start by configuring the network scheme. For this test, we will also add some latency so we can check with the RTT (round-trip time) if Testground is successfully applying network conditions!

```go
import (
  "time"
  "github.com/testground/sdk-go/network"
)

// ...

ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
defer cancel()

client := initCtx.SyncClient
netclient := initCtx.NetClient

config := &network.Config{
  Network: "default", // Control the "default" network.
  Enable: true,
  Default: network.LinkShape{
    Latency:   100 * time.Millisecond,
    Bandwidth: 1 << 20, // 1Mib
  },
  CallbackState: "network-configured",
  RoutingPolicy: network.DenyAll,
}

runenv.RecordMessage("before netclient.MustConfigureNetwork")
netclient.MustConfigureNetwork(ctx, config)
```

Now, what we are going to do, is to define custom IPs inside the network for each of the instaces. For that, we first signal that we are going to start the "ip allocation" phase and that will make all instances hold until they're all in the same part of the code. The signalling returns a sequence number that we are going to use here, as a convinience, to attribute our IP.

```go
seq := client.MustSignalAndWait(ctx, "ip-allocation", runenv.TestInstanceCount)
runenv.RecordMessage("I am %d", seq)
```

Imagining an IP formatted as `A.B.C.D`, we are only going to define the C and the D part. To get the same value for the C on all instances, we apply the following bitwise operations. The D part is simply our sequence number.

```go
ipC := byte((seq >> 8) + 1)
ipD := byte(seq)
```

Now, let's actually configure the network with the new IP values. For this, we will also start a TCP listener if we're the number one. It doesn't matter if it's the number one, or number two, as long as the entire code is consistent. We need this TCP listener, so that the other instance will connect to it afterwards! 

```go
// We define the new settings!
config.IPv4 = runenv.TestSubnet
config.IPv4.IP = append(config.IPv4.IP[0:2:2], ipC, ipD)
config.CallbackState = "ip-changed"

// Declare some variables...
var (
  listener *net.TCPListener
  conn     *net.TCPConn
  err      error
)

// Start the TCP listener if we're seq == 1
if seq == 1 {
  listener, err = net.ListenTCP("tcp4", &net.TCPAddr{Port: 1234})
  if err != nil {
    return err
  }
  defer listener.Close()
}

// Configure the network with the new configurationn!
runenv.RecordMessage("before reconfiguring network")
netclient.MustConfigureNetwork(ctx, config)
```

After this point, both instances should do different things. Thus, we will use a switch statement and the sequence number to distinguish between them. While the instance 1 starts accepting a TCP connection, the instance 2 tries to connect to it!

```go
switch seq {
case 1:
  conn, err = listener.AcceptTCP()
case 2:
  conn, err = net.DialTCP("tcp4", nil, &net.TCPAddr{
    IP:   append(config.IPv4.IP[:3:3], 1),
    Port: 1234,
  })
default:
  return fmt.Errorf("expected at most two test instances")
}
if err != nil {
  return err
}

defer conn.Close()

// trying to measure latency here.
err = conn.SetNoDelay(true)
if err != nil {
  return err
}
```

Now that both instances are connected to each other, we will make sure that both of them are ready, by writing and reading from the socket. This way we make sure there's a connection adn that it is working.

```go
runenv.RecordMessage("waiting until ready")
buf := make([]byte, 1)

// Wait till both sides are ready
_, err = conn.Write([]byte{0})
if err != nil {
  return err
}

_, err = conn.Read(buf)
if err != nil {
  return err
}
```

We're almost in the end! Let's do the actual ping pong now, and measure the RTT! For this, each instance writes their own ID, then the other instance receives it and returns it. In the end, the ID we got should be the same as our sequence number. If it's not, something went wrong down the wire! 💥

```go
start := time.Now()

// write sequence number.
runenv.RecordMessage("writing my id")
_, err = conn.Write([]byte{byte(seq)})
if err != nil {
  return err
}

// pong other sequence number
runenv.RecordMessage("reading their id")
_, err = conn.Read(buf)
if err != nil {
  return err
}

runenv.RecordMessage("returning their id")
_, err = conn.Write(buf)
if err != nil {
  return err
}

runenv.RecordMessage("reading my id")
// read our sequence number
_, err = conn.Read(buf)
if err != nil {
  return err
}

runenv.RecordMessage("done")

// stop
end := time.Now()

// check the sequence number.
if buf[0] != byte(seq) {
  return fmt.Errorf("read unexpected value")
}
```

Now, let's check if the RTT is correct. As you might've noticed, we stored the start and end timings of the ping pong activity. Now, we calculate the minimum RTT, which is the latency times two, and a maximum RTT just to give it a bit of room to breath. I set it to 50 milliseconds, which is a lot, but you can play with it to see!

```go
rttMin := config.Default.Latency * 2
rttMax := rttMin + 50*time.Millisecond

// check the RTT
rtt := end.Sub(start)
if rtt < rttMin || rtt > rttMax {
  return fmt.Errorf("expected an RTT between %s and %s, got %s", rttMin, rttMax, rtt)
}

runenv.RecordMessage("ping RTT was %s [%s, %s]", rtt, rttMin, rttMax)
return nil
```

You now have a full functional ping pong test that Testground can run! To do so, just run the following command:

```bash
$ testground run single \
    --builder docker:go \
    --runner local:docker \
    -i 2 --plan my-plan --testcase ping-pong --wait
```

After waiting a few seconds, for the Docker build build, the test will start running. You will be able to see some _very colorful_ output on your screen. In the end, it should look something similar to this:

```
MESSAGE << single[000] (a5d263) >> waiting until ready
MESSAGE << single[001] (28ddad) >> waiting until ready
MESSAGE << single[001] (28ddad) >> writing my id
MESSAGE << single[001] (28ddad) >> reading their id
MESSAGE << single[000] (a5d263) >> writing my id
MESSAGE << single[000] (a5d263) >> reading their id
MESSAGE << single[000] (a5d263) >> returning their id
MESSAGE << single[000] (a5d263) >> reading my id
MESSAGE << single[001] (28ddad) >> returning their id
MESSAGE << single[001] (28ddad) >> reading my id
MESSAGE << single[001] (28ddad) >> done
MESSAGE << single[001] (28ddad) >> ping RTT was 207.835199ms [200ms, 250ms]
MESSAGE << single[000] (a5d263) >> done
MESSAGE << single[000] (a5d263) >> ping RTT was 208.659726ms [200ms, 250ms]
     OK << single[000] (a5d263) >>
     OK << single[001] (28ddad) >>
```

## To conclude 🧶

*Et voilá!* You just built a simple test plan for Testground that successfully exercises it capabilities of shaping the network, as well as inter-instance connectivity. Testground, despite being in its early stages, is already being used by some of our projects to run tests.

We're now working on TaaS, [Testground as a Service](https://github.com/testground/testground/issues/1148), something we've wanted to do for some time. Having Testground running as a service will allow us to easily integrate it with CIs or even having online dashboards to check tests results and analyse data.

If you're interested, don't hesitate to [read more](https://github.com/testground/testground) about the project!