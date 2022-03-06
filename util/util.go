package util //store common functions for use by servers and client
import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/mipsmonsta/chunky/chunky"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// For Clients to use
func SendChunks(absFilePath string, stream *chunky.ChunkUploadService_UploadClient) {

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

	var p []byte = make([]byte, 1024)

	for {
		n, err := dataReader.Read(p)

		if n == 0 && err == io.EOF {
			
			log.Printf("Completed sending file...%q\n", fileName)
			break
		}
		bContent := chunky.Chunk_Content{
			Content: p[:n],
		}
		c := chunky.Chunk{
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

//For servers to use
var (
	File_Dir = "./files"
)

type DoSomething struct {
	Function func(string)
}

// hook for processing the file path
var DefaultSomething DoSomething = DoSomething{Function: nil}

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
	filePath := File_Dir + "/" + base_filename
	f, err := os.Create(File_Dir + "/" + base_filename)
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

	if DefaultSomething.Function != nil {
		DefaultSomething.Function(filePath)
	}

	return nil
}

func RegisterServices(s *grpc.Server){
	chunky.RegisterChunkUploadServiceServer(s, &chunkyService{})
}

func CreateFilesDir(customFileDir *string){
	if customFileDir != nil {
		File_Dir = *customFileDir
	}
	if _, err := os.Stat(File_Dir); os.IsNotExist(err) {
		_ = os.Mkdir(File_Dir, 0700)
	}
}