.PHONY: installprotocgorums
installprotocgorums:
	@echo installing protoc-gen-gorums with gorums linked...
	@go install github.com/relab/gorums/cmd/protoc-gen-gorums

.PHONY: proto
proto: installprotocgorums
	protoc -I=$(GOPATH)/src/:. --gorums_out=plugins=grpc+gorums:. byzq.proto

.PHONY: bench 
bench:
	go test -run=NONE -benchmem -benchtime=5s -bench=.
