package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"

	"github.com/mipsmonsta/chunky/chunky"
	"github.com/mipsmonsta/chunky/util"

	"log"
	"os"
	"path/filepath"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	Port=":50051"
)

func setupGRpcConnection(addr string) (*grpc.ClientConn, error) {
	return grpc.DialContext(
		context.Background(), 
		addr, 
		grpc.WithTransportCredentials(insecure.NewCredentials()), 
		grpc.WithBlock(), 
	)
}

func getChunkUploadServiceClient(conn *grpc.ClientConn) chunky.ChunkUploadServiceClient{
	return chunky.NewChunkUploadServiceClient(conn)
}

var file_path string
var addr_ip net.IP
func setupFlags(){
	//create a new flag set
	fs := flag.NewFlagSet("ChunkUpload", flag.ExitOnError)
	fs.SetOutput(os.Stdout)
	
	fs.Func("ip", "IP address of Server", func(s string) error {

		if ip := net.ParseIP(s); ip != nil{
			addr_ip = ip
			return nil
		}

		return errors.New("ip address is invalid")

	})

	//do validate of the file flag value
	fs.Func("file", "File path to upload", func(s string) error {
		//check file exists
		if _, err := os.Stat(s); err == nil {
			file_path = s
			return nil
		} 
		return errors.New("file to upload does not exists")
	})

	
	fs.Parse(os.Args[1:])

	// won't interfere if <command> -h[--help]
	if file_path == "" || addr_ip == nil {// if --file is not set, invoke Usage
		fs.Usage() //cannot use PrintDefaults() a
				   //s the "Usage ..." won't show
		os.Exit(1) //exit 
	}
		
}



func init(){
	//set up flags
	setupFlags()
}

func main(){

	conn, err := setupGRpcConnection(addr_ip.String() + Port)
	if err != nil {
		fmt.Println(fmt.Errorf("error in setting up Grpc connection: %w", err))
		os.Exit(1)
	}

	chunkUploadClient := getChunkUploadServiceClient(conn)
	stream, err := chunkUploadClient.Upload(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	
	defer func(){
		if err:= recover(); err != nil {
			fmt.Println(err)
			_, _ = stream.CloseAndRecv()
		}

	}()
	

	absFilePath, err := filepath.Abs(file_path)
	if err != nil {
		log.Fatal(err)
	}

	util.SendChunks(absFilePath, &stream)
	
}