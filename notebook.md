# notebook example

> These are some hello world examples you can run from Notes

## Bash

This will output a hello world program

```bash
echo hello world
```

## Python 3

This is a python hello world program

```python
print("hello world")
```

## Golang

This is a Go hello world one liner

```go
fmt.Println("hello world!!")
```

## Shell scripts again

```sh
echo "1
2
3"
```

## Python again

This is a Python hello world program

```python
print("hello world?")

```

## Go again

This is a Go hello world program

```go
package main

func main() {
    fmt.Println("hello world??")
}
```

## Generic code block

This would be autodetected by Javascript

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

## Dockerfile (build + run output)

This one shows output during build (`RUN`) and then output when the container starts (`CMD`)

```dockerfile
FROM alpine:3.20

RUN echo "hello world from build step"
CMD ["echo", "hello world from container start"]
```
