# byzq
Byzantine Quorum Protocol

// Byzantine Safe Register, ref. Algo. 4.14 in RSDP.
// Requires authenticated channels
// RequestID field of messages not needed since gRPC handles request matching.


// Authenticated-Data Byzantine Quorum, ref. Algo. 4.15 in RSDP.
// Requires authenticated channels
// RequestID field of messages not needed since gRPC handles request matching.


## Running localhost example 

#### Start servers

```shell
cd cmd/byzserver
./startbyzq5.sh
```

#### Start a client

```shell
cd cmd/byzclient
go build
./byzclient
```

## Quorum function benchmarks (Maybe add this later)

## TODO : benchmarks and tests

```make bench```
