package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/Benchamon/dc-final/api"
	"github.com/Benchamon/dc-final/controller"
	"github.com/Benchamon/dc-final/scheduler"
)

func main() {
	log.Println("Welcome to the Distributed and Parallel Image Processing System")

	// Start Controller
	go controller.Start()

	// Start Scheduler
	jobs := make(chan scheduler.Job)
	go scheduler.Start(jobs)
	// Send sample jobs
	sampleJob := scheduler.Job{Address: "localhost:50051", RPCName: "hello"}

	// API
	go api.Start()

	for {
		sampleJob.RPCName = fmt.Sprintf("hello-%v", rand.Intn(10000))
		jobs <- sampleJob
		time.Sleep(time.Second * 5)
	}
}
