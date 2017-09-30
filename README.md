# byzq - Byzantine Quorum Protocol

#### Authenticated-Data Byzantine Quorum.
* Ref. Algo. 4.15 in RSDP.
* Requires authenticated channels
* RequestID field of messages not needed since gRPC handles request matching.

## Running localhost example 

#### Start four servers

```shell
cd cmd/byzserver
./startbyzq4.sh
```

#### Start a writer client (should be started first so that server has data for the reader client)

```shell
cd cmd/byzclient
go build
./byzclient -writer
```

#### Start a reader client

```shell
cd cmd/byzclient
go build
./byzclient 
```

## Quorum function benchmarks

```make bench```
