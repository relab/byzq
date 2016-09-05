PROTOC_PLUGIN 			:= gorums_out

.PHONY: installprotocgorums
installprotocgorums:
	@echo installing protoc-gen-gorums with gorums linked...
	@go install github.com/relab/gorums/cmd/protoc-gen-gorums

.PHONY: proto
proto: installprotocgorums
	protoc --$(PROTOC_PLUGIN)=plugins=grpc+gorums:. byzq.proto

.PHONY: bench 
bench:
	go test github.com/relab/byzq/cmd/byzclient -run=NONE -benchmem -benchtime=5s -bench=.
