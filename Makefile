PROTOC_PLUGIN 			:= gorums_out

PROTO_PKG 			:= proto
PROTO_GQRPC_PKG_RPATH 		:= $(PROTO_PKG)/byzq

.PHONY: installprotocgorums
installprotocgorums:
	@echo installing protoc-gen-gorums with gorums linked...
	@go install github.com/relab/gorums/cmd/protoc-gen-gorums

.PHONY: proto
proto: installprotocgorums
	protoc --$(PROTOC_PLUGIN)=plugins=grpc+gorums:. $(PROTO_GQRPC_PKG_RPATH)/byzq.proto

.PHONY: bench 
bench:
	go test github.com/relab/byzq/cmd/byzclient -run=NONE -benchmem -benchtime=5s -bench=.
