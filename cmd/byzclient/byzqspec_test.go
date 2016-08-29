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

// run tests with: go test -v

// run benchmarks: go test -run=$$ -benchmem -benchtime=5s -bench=.

func TestMain(m *testing.M) {
	silentLogger := log.New(ioutil.Discard, "", log.LstdFlags)
	grpclog.SetLogger(silentLogger)
	grpc.EnableTracing = false
	res := m.Run()
	os.Exit(res)
}

var byzQTests = []struct {
	n   int
	f   int // expected value
	q   int // expected value
	err string
}{
	{4, 0, 2, "Byzantine masking quorums require n>4f replicas; only got n=4, yielding f=0"},
	{5, 1, 3, ""},
	{6, 1, 4, ""},
	{7, 1, 4, ""},
	{8, 1, 5, ""},
	{9, 2, 6, ""},
	{10, 2, 7, ""},
	{11, 2, 7, ""},
	{12, 2, 8, ""},
	{13, 3, 9, ""},
	{14, 3, 10, ""},
}

func TestByzQ(t *testing.T) {
	for _, test := range byzQTests {
		bq, err := NewByzQ(test.n)
		if err != nil {
			if err.Error() != test.err {
				t.Errorf("got '%v', expected '%v'", err.Error(), test.err)
			}
			continue
		}
		if bq.f != test.f {
			t.Errorf("got f=%d, expected f=%d", bq.f, test.f)
		}
		if bq.q != test.q {
			t.Errorf("got q=%d, expected q=%d", bq.q, test.q)
		}
	}

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
	{
		"base-case quorum",
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
		"approx. worst-case quorum",
		[]*byzq.State{
			{Timestamp: 1, Value: 2},
			{Timestamp: 2, Value: 4},
			{Timestamp: 1, Value: 5},
			myVal,
			myVal,
		},
		myVal,
		true,
	},
}

func TestByzReadQF(t *testing.T) {
	qspec, err := NewByzQ(5)
	if err != nil {
		t.Error(err)
	}
	for _, test := range byzReadQFTests {
		t.Run("ByzQ(5,1)-"+test.name, func(t *testing.T) {
			reply, byzquorum := qspec.ReadQF(test.replies)
			if byzquorum != test.rq {
				t.Errorf("got %t, want %t", byzquorum, test.rq)
			}
			if !reply.Equal(test.expected) {
				t.Errorf("got %d, want %d as quorum reply", reply, test.expected)
			}
		})
	}
}

func BenchmarkByzReadQF(b *testing.B) {
	qspec, err := NewByzQ(5)
	if err != nil {
		b.Error(err)
	}
	for _, test := range byzReadQFTests {
		if !strings.Contains(test.name, "case") {
			continue
		}
		b.Run("ByzQ(5,1)-"+test.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				qspec.ReadQF(test.replies)
			}
		})
	}
}
