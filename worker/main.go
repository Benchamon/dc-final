package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	pb "github.com/Benchamon/dc-final/proto"
	"go.nanomsg.org/mangos"
	"google.golang.org/grpc"
	"go.nanomsg.org/mangos/protocol/respondent"
	"github.com/Benchamon/dc-final/controller"
	_ "go.nanomsg.org/mangos/transport/all"
)

var defaultRPCPort = 50051
type server struct {
	pb.UnimplementedTaskServer
}

var (
	controllerAddress = ""
	WorkerName        = ""
	tags              = ""
	status            = ""
	workDone          = 0
	usage             = 0
	port              = 0
	jobsDone          = 0
)

func die(format string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("RPC: Received: %v", in.GetName())
	if in.GetName() == "test" {
		workDone += 1
		log.Printf("RPC [Worker] %+v: running", WorkerName)
		usage += 1
		status = "Running"
		usage -= 1
		return &pb.HelloReply{Message: "Hello, " + WorkerName + " running"}, nil
	} else {
		workDone += 1
		log.Printf("[Worker] %+v: calling", WorkerName)
		usage += 1
		status = "Running"
		return &pb.HelloReply{Message: "Hello " + WorkerName}, nil
	}	
}

//no sirve profe :(
func (s *server) FilterImage(ctx context.Context, in *pb.ImgRequest) (*pb.ImgReply, error) {

	msg := fmt.Sprintf("Filtering image: %v filter: %v \n", in.GetImg().Filepath, in.GetImg().Filter)
	fmt.Printf(msg)
	controller.UpdateWorkerStatus(WorkerName, "busy")
	newFilename := "new file"

	if in.GetImg().Filter == "grayscale" {
		newFilename = fmt.Sprintf("f%v_%v", in.Img.Index, in.Img.Name) + ".png"
	} else {
        return &pb.ImgReply{Message: "not supported " + WorkerName}, nil
	}
	controller.UpdateUsage(WorkerName)
	controller.UpdateWorkerStatus(WorkerName, "free")
	return &pb.ImgReply{Message: fmt.Sprintf("%v=%v", newFilename, in.Img.Workload)}, nil
}

func init() {
	flag.StringVar(&controllerAddress, "controller", "tcp://localhost:40899", "Controller address")
	flag.StringVar(&WorkerName, "worker-name", "hard-worker", "Worker Name")
	flag.StringVar(&tags, "tags", "gpu,superCPU,largeMemory", "Comma-separated worker tags")
}

func joinCluster() {
	var sock mangos.Socket
	var err error
	var msg []byte

	if sock, err = respondent.NewSocket(); err != nil {
		die("no socket: %s", err.Error())
	}

	log.Printf("Connecting to: %s", controllerAddress)
	if err = sock.Dial(controllerAddress); err != nil {
		die("can't dial: %s", err.Error())
	}
	for {
		if msg, err = sock.Recv(); err != nil {
			die("Error in recv function: %s", err.Error())
		}
		info := fmt.Sprintf("%v %v %v %v %v %v", WorkerName, status, usage, tags, defaultRPCPort, jobsDone)
		if err = sock.Send([]byte(info)); err != nil {
			die("Error sending: %s", err.Error())
		}
		log.Printf("Message-Passing: Worker(%s): Received %s\n", WorkerName, string(msg))
	}
}

func getAvailablePort() int {
	port := defaultRPCPort
	for {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
		if err != nil {
			port = port + 1
			continue
		}
		ln.Close()
		break
	}
	return port
}

func main() {
	flag.Parse()
	go joinCluster()
	rpcPort := getAvailablePort()
	defaultRPCPort = rpcPort
	log.Printf("Starting RPC Service on localhost:%v", rpcPort)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", rpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterTaskServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}