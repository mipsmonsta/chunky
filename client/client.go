package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"grpfile/chunky"
	"io"
	"log"
	"os"
	"path/filepath"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func setupFlags(){
	//create a new flag set
	fs := flag.NewFlagSet("ChunkUpload", flag.ExitOnError)
	fs.SetOutput(os.Stdout)
	
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
	if file_path == "" {// if --file is not set, invoke Usage
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

	conn, err := setupGRpcConnection(":50051")
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

	sendChunks(absFilePath, &stream)
	
}


func sendChunks(absFilePath string, stream *chunky.ChunkUploadService_UploadClient) {

	fileName := filepath.Base(absFilePath)

	fn := chunky.Chunk_FileName{
		FileName: fileName,
	}
	c := chunky.Chunk{
		Body: &fn,
	}

	err := (*stream).Send(&c)
	if err != nil {
		panic(err)
	}

	f, err := os.Open(absFilePath)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	dataReader := bufio.NewReader(f)

	for {
		b, err := dataReader.ReadByte()
		if err == io.EOF {
			log.Printf("Completed sending file...%q\n", fileName)
			break
		}
		bContent := chunky.Chunk_Content{
			Content: []byte{b},
		}
		c:= chunky.Chunk{
			Body: &bContent,
		}
		err = (*stream).Send(&c)
		if err != nil {
			panic(err)
		}
	}

	_, err = (*stream).CloseAndRecv()
	if err != nil {
		fmt.Println(err)
	}
}