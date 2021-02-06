---
publishDate: "2017-09-08T00:00:00.000Z"
tags:
- server
- web
- go
title: How to create a web server in Go
---

Go is a relatively new player in the world of programming, having been around since 2009. It was created by Google and many open-source contributors.

It has recently reached version 1.9 and 2.0 is on [its way](https://blog.golang.org/toward-go2). Despite having only 8 years, its popularity has been increasing exponentially, mainly on networking and web server fields.

<!--more-->

{{< figure
  src="graph.jpg"
  caption="TIOBE Index for September 2017"
  alt="TIOBE Index for September 2017" >}}

Go is an incredible effective language to create web servers and web services thanks to its powerful standard library that can help us to do whatever we want with the web.
Start by [downloading](https://golang.org/doc/install) and installing Go on your computer. It's pretty straightforward. You can check if it was correctly installed by running go version on your console.

I'll now give you two different examples of servers: the first one will print a "Hello {your name}" message and the second one will also serve files from the file system.

### #1 Say Hello Server

Now, create a folder inside `$GOPATH/src` called server and that folder is everything we will be using in this brief example. Make a `main.go` file inside the folder you just created and paste the following code:

```go
package main

import (
  "net/http"
  "strings"
)

func sayHello(w http.ResponseWriter, r *http.Request) {
  message := r.URL.Path
  message = strings.TrimPrefix(message, "/")
  message = "Hello " + message
  w.Write([]byte(message))
}

func main() {
  http.HandleFunc("/", sayHello)
  if err := http.ListenAndServe(":8080", nil); err != nil {
    panic(err)
  }
}
```

Then, open the command line and run `go run main.go`. If you now go to [`localhost:8080/George`](http://localhost:8080/George), you'll see the message "Hello George" printed out.

In this little code, we are creating a handler called `sayHello` which retrieves the path of the URL (first line), removes the first slash (second line) and appends the "Hello" to the beginning of the sentence. Then, we write the final message to the `ResponseWriter` converted to bytes.

The main function is also easy to understand: in the first line we tell the server to use the handler `sayHello` to every request that hits the server and in the second and third lines we start the server on port 8080 and handle an error if something wrong happens.

### #2 File System Server

Here is just another example, a lot simpler, where we serve static files from the file system (`src` folder) on every path except `/ping`, which will be handled by `ping` handler.

```go
package main

import (
  "net/http"
)

func ping(w http.ResponseWriter, r *http.Request) {
  w.Write([]byte("pong"))
}

func main() {
  http.Handle("/", http.FileServer(http.Dir("./src")))
  http.HandleFunc("/ping", ping)
  if err := http.ListenAndServe(":8080", nil); err != nil {
    panic(err)
  }
}
```