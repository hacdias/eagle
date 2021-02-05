---
title: gRPC
---

**gRPC** (gRPC Remote Procedure Calls) is a modern [Remote Procedure Call](/notes/remote-procedure-call/) mechanism, similar to traditional RPC, but developed with cloud services in mind. It was started  in 2015 by Google and it is an open source project.

This implementation is built on top of [HTTP/2](/notes/http/), [TLS](/notes/tls/) and [TCP](/notes/tcp/) protocols.  Besides, it provides implementations for many programming languages.

For its [IDL](Interface%20Description%20Language), it uses [Protocol Buffers](/notes/protocol-buffers/). The lower level layers of gPRC do not depend on the IDL so, in theory, it would be possible to use other alternatives to protocol buffers.

## "Hello World" example

Protocol Buffer definition:

```protobuf
message HelloRequest {
  string name = 1;
  repeated string hobbies = 2;
}

message HelloResponse {
  string greeting = 1;
}

service HelloWorldService {
  rpc greeting(HelloRequest) returns (HelloResponse);
}
```

Server code (Java):

```java
public class HelloServer {
  public static void main(String[] args) throws Exception {
    // ...
    final int port = Integer.parseInt(args[0]);
    final BindableService impl = new HelloWorldServiceImpl();

    Server server = ServerBuilder
      .forPort(port)
      .addService(impl)
      .build();

    server.start();
    server.awaitTermination();
  }
}

public class HelloWorldServiceImpl extends HelloServiceGrpc.HelloServiceImplBase {

  @Override
  public void greeting(Hello.HelloRequest request, StreamObserver<Hello.HelloResponse> responseObserver) {
    Hello.HelloResponse response = Hello.HelloResponse
      .newBuilder()
      .setGreeting("Hello " + request.getName())
      .build();

    responseObserver.onNext(response);
    responseObserver.onCompleted();
  }
}
```

Client code (Java):

```java
final ManagedChannel channel = ManagedChannelBuilder
  .forTarget(target)
  .usePlainText() // Unsecured channel
  .build();

HelloServiceGrpc.HelloServiceBlockingStub stub = HelloServiceGrpc.newBlockingStub(channel);

Hello.HelloRequest request = Hello.HelloRequest.newBuilder()
  .setName("friend")
  .build();

Hello.HelloResponse response = stub.greeting(request);

System.out.println(response);
channel.shutdownNow();
```