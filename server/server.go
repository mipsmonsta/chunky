package main

import (
	"fmt"
	"grpfile/chunky"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type chunkyService struct{
	chunky.UnimplementedChunkUploadServiceServer
}

func (c *chunkyService) Upload(stream chunky.ChunkUploadService_UploadServer) error {
	var data []byte
	var base_filename string
	var uStatus chunky.UploadStatus
	for {
		r, err := stream.Recv()
		if err == io.EOF {
			break
		}
		
		switch t:= r.Body.(type) {
		case *chunky.Chunk_FileName:
			base_filename = r.GetFileName()
		case *chunky.Chunk_Content:
			b := r.GetContent()
			data = append(data, b...)
		case nil:
			return status.Error(codes.InvalidArgument, "Message doesn't contain fileName or Content")
		default:
			
			return status.Errorf(codes.FailedPrecondition, "Unexpected message type: $s", t)
		}
	}
	//check filename is just base i.e. last path item
	pathList := filepath.SplitList(base_filename)
	if len(pathList) > 1 {
		return status.Error(codes.Internal, "filename is not just base name")
	}

	//save data
	f, err := os.Create("../files/" + base_filename)
	if err != nil {
		return status.Error(codes.Internal, "file not created")
	}
	defer f.Close()
	f.Write(data)
	
	uStatus = chunky.UploadStatus{
		Message: "Data received",
		Code: chunky.UploadStatusCode_OK,
	}
	err = stream.SendAndClose(&uStatus)
	if err != nil {
		return err
	}
	return nil
}

func registerServices( s *grpc.Server){
	chunky.RegisterChunkUploadServiceServer(s, &chunkyService{})
}

func main() {
	l, err := net.Listen("tcp", ":50051")
	if err != nil {
		e := fmt.Errorf("TCP Error: %w", err)
		fmt.Println(e)
		os.Exit(1)
	}

	s := grpc.NewServer()
	registerServices(s)
	log.Fatal(s.Serve(l))

}