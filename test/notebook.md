# Notebook Example

> These are some hello world examples you can run from Notes

## Shell scripts

This will output a hello world program

### bash

```bash
git status
```

### sh

```sh
echo "123"
```

### murex

```murex
echo bob -> grep bob
```

## Python 3

This is a python hello world program

```python
print("hello world")
```

## Golang

This is a Go hello world one-liner

```go
log.Println("hello world")
```

And here is a complete program

```go
package main

func main() {
    fmt.Println("hello world")

    for i := range 5 {
        time.Sleep(1 * time.Second)
        fmt.Println(i+1)
    }

    fmt.Println("goodbye cruel world")
}
```

## Generic code block

This should be autodetected as Perl

```
use warnings;
print "hello world";
```

## Dockerfile

This builds a tiny image that prints hello world when run

```dockerfile
FROM alpine:3.20

CMD ["echo", "hello world from docker"]
```

This one shows output during build (`RUN`) and then output when the container starts (`CMD`)

```dockerfile
FROM alpine:3.20

RUN echo "hello world from build step"
CMD ["echo", "hello world from container start"]
```

## C

This is a C hello world program

```c
#include <stdio.h>
#include <unistd.h>

int main() {
    printf("hello world!\n");
    sleep(2);
    printf("goodbye cruel world\n");
    return 0;
}
```

And this is a one-liner

```c
printf("hello world!\n");
```

## C++

This is a C++ hello world program

```cpp
#include <iostream>
#include <chrono>
#include <thread>

int main() {
    std::cout << "hello world!!" << std::endl;
    
    for (int i = 0; i < 5; i++) {
        std::this_thread::sleep_for(std::chrono::seconds(1));
        std::cout << (i + 1) << std::endl;
    }
    
    std::cout << "goodbye cruel world" << std::endl;
    return 0;
}
```

And this is a one-liner

```cpp
std::cout << "hello world!!" << std::endl;
```

## JavaScript

This is a JavaScript hello world program

```javascript
console.log("hello world!");

for (let i = 1; i <= 5; i++) {
    setTimeout(() => {
        console.log(i);
    }, i * 1000);
}

setTimeout(() => {
    console.log("goodbye cruel world");
}, 6000);
```

## TypeScript

This is a TypeScript hello world program

```typescript
function greet(message: string): void {
    console.log(message);
}

greet("hello world!!");

for (let i = 1; i <= 5; i++) {
    setTimeout(() => {
        console.log(i);
    }, i * 1000);
}

setTimeout(() => {
    greet("goodbye cruel world");
}, 6000);
```
