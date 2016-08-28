package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/relab/byzq/proto/byzq"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

func TestMain(m *testing.M) {
	silentLogger := log.New(ioutil.Discard, "", log.LstdFlags)
	grpclog.SetLogger(silentLogger)
	grpc.EnableTracing = false
	res := m.Run()
	os.Exit(res)
}

const val = 42

var myVal = &byzq.State{Timestamp: 3, Value: val}

var byzReadQFTests = []struct {
	name     string
	replies  []*byzq.State
	expected *byzq.State
	rq       bool
}{
	{
		"nil input",
		nil,
		nil,
		false,
	},
	{
		"len=0 input",
		[]*byzq.State{},
		nil,
		false,
	},
	{
		"no quorum (I)",
		[]*byzq.State{
			{Timestamp: 1, Value: 2},
			{Timestamp: 1, Value: val},
		},
		nil,
		false,
	},
	{
		"no quorum (II)",
		[]*byzq.State{
			{Timestamp: 1, Value: 2},
			{Timestamp: 1, Value: val},
			{Timestamp: 1, Value: val},
		},
		nil,
		false,
	},
	{
		"no quorum (III); default value",
		[]*byzq.State{
			{Timestamp: 1, Value: 2},
			{Timestamp: 1, Value: 3},
			{Timestamp: 1, Value: 4},
			{Timestamp: 1, Value: val},
		},
		&defaultVal,
		true,
	},
	{
		"no quorum (IV); default value",
		[]*byzq.State{
			{Timestamp: 1, Value: 2},
			{Timestamp: 1, Value: 3},
			{Timestamp: 2, Value: val},
			{Timestamp: 3, Value: val},
			{Timestamp: 1, Value: 4},
		},
		&defaultVal,
		true,
	},
	{
		//todo: decide if #replies > n should be accepted ?
		"no quorum (V); default value",
		[]*byzq.State{
			{Timestamp: 1, Value: 2},
			{Timestamp: 2, Value: 3},
			{Timestamp: 3, Value: val},
			{Timestamp: 1, Value: val},
			{Timestamp: 2, Value: 2},
			{Timestamp: 3, Value: 4},
		},
		&defaultVal,
		true,
	},
	{
		"quorum (I)",
		[]*byzq.State{
			myVal,
			myVal,
			myVal,
			myVal,
		},
		myVal,
		true,
	},
	{
		"quorum (II)",
		[]*byzq.State{
			{Timestamp: 1, Value: 2},
			myVal,
			myVal,
			myVal,
		},
		myVal,
		true,
	},
	{
		"quorum (III)",
		[]*byzq.State{
			myVal,
			myVal,
			myVal,
			myVal,
			myVal,
		},
		myVal,
		true,
	},
	{
		"quorum (IV)",
		[]*byzq.State{
			{Timestamp: 1, Value: 2},
			{Timestamp: 1, Value: 2},
			myVal,
			myVal,
			myVal,
		},
		myVal,
		true,
	},
}

var qspecs = []struct {
	name string
	spec byzq.QuorumSpec
}{
	{"ByzQ(5,1)", NewByzQ(5, 1)},
	// {"ByzQ(9,2)", NewByzQ(9, 2)},
}

func TestByzReadQF(t *testing.T) {
	for _, qspec := range qspecs {
		for _, test := range byzReadQFTests {
			t.Run(qspec.name+"-"+test.name, func(t *testing.T) {
				reply, byzquorum := qspec.spec.ReadQF(test.replies)
				if byzquorum != test.rq {
					t.Errorf("got %t, want %t", byzquorum, test.rq)
				}
				if !reply.Equal(test.expected) {
					t.Errorf("got %d, want %d as quorum reply", reply, test.expected)
				}
			})
		}
	}
}

func BenchmarkByzReadQF(b *testing.B) {
	for _, qspec := range qspecs {
		for _, test := range byzReadQFTests {
			if !strings.Contains(test.name, "case") {
				continue
			}
			b.Run(qspec.name+"-"+test.name, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					qspec.spec.ReadQF(test.replies)
				}
			})
		}
	}
}
