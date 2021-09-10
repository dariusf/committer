package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/vadiminshakov/committer/peer"
	pb "github.com/vadiminshakov/committer/proto"
	"github.com/vadiminshakov/committer/trace"
)

const addr = "localhost:3000"

func main() {
	start := time.Now()
	tracer, err := trace.Tracer("client", addr)
	if err != nil {
		panic(err)
	}
	cli, err := peer.New(addr, tracer)
	if err != nil {
		panic(err)
	}
	reqs, _ := os.LookupEnv("REQUESTS")
	requests, _ := strconv.Atoi(reqs)

	for i := 0; i < requests; i++ {
		resp, err := cli.Put(context.Background(), "key"+strconv.Itoa(i), []byte("req"+strconv.Itoa(i)))
		if err != nil {
			panic(err)
		}
		if resp.Type != pb.Type_ACK {
			panic("msg is not acknowledged")
		}
	}

	fmt.Printf("Total time taken: %d\n", time.Since(start).Nanoseconds())

	// read committed keys
	//key, err := cli.Get(context.Background(), "key3")
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println(string(key.Value))
}
