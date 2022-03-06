package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/mipsmonsta/chunky/util"
	"google.golang.org/grpc"
)



func main() {
	l, err := net.Listen("tcp", ":50051")
	if err != nil {
		e := fmt.Errorf("TCP Error: %w", err)
		fmt.Println(e)
		os.Exit(1)
	}

	util.CreateFilesDir(nil)

	s := grpc.NewServer()
	util.RegisterServices(s)
	log.Fatal(s.Serve(l))

}