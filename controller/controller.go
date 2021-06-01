package controller
import (
	"fmt"
	"os"
	"time"
	"strings"
	"strconv"
	"go.nanomsg.org/mangos"
	"go.nanomsg.org/mangos/protocol/surveyor"
	_ "go.nanomsg.org/mangos/transport/all"
)

var controllerAddress = "tcp://localhost:40899"
var sock mangos.Socket
var done = make(chan string)
var Uploads = make(map[string]Image)
var actions = make(map[string]Action)
var Workers = make(map[string]Worker)
var Workloads = make(map[string]Workload)
var filters = make(map[string]ImageService)

type Worker struct {
	Name     string `json:"name"`
	Tags     string `json:"tags"`
	Status   string `json:"status"`
	Usage    int    `json:"usage"`
	URL      string `json:"url"`
	Active   bool   `json:"active"`
	Port     int    `json:"port"`
	JobsDone int    `json:"jobsDone"` 
}

type ImageService struct{
	Id int
	Image string
	Workload string
}
type Workload struct{
	Id string
	Filter string
	Name string
	Status string
	Jobs int
	Imgs []string
	Filtered []string
}
type Image struct {
	Id int
	Name string
	Ext string
}
type Action struct {
	id		int
	worker 	string
}

func die(format string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func date() string {
	return time.Now().Format(time.ANSIC)
}

func Start() {
	var sock mangos.Socket
	var err error
	if sock, err = surveyor.NewSocket(); err != nil {
		die("Cant get socket: %s", err)
	}
	if err = sock.Listen(controllerAddress); err != nil {
		die("Cant listen socket: %s", err.Error())
	}
	err = sock.SetOption(mangos.OptionSurveyTime, time.Second)
	if err != nil {
		die("Error: %s", err.Error())
	}
	
	var resp []byte
	for {
		err = sock.Send([]byte("Welcome"))
		if err != nil {
			die("No worker %+v", err.Error())
		}
		for {
			if resp, err = sock.Recv(); err != nil {
				break
			}
			exists := false
			worker := GetWorkerInfo(string(resp))
			for _, w := range Workers {
				if w.Name == worker.Name {
					exists = true
				}
			}
			if !exists {
				Workers[worker.Name] = worker
			}
			PrintWorker(worker)
		}
	}
}

func PrintWorker(worker Worker) {
	fmt.Println(Workers[worker.Name].Name, " serves in localhost:", Workers[worker.Name].Port, "\n")
}

func GetWorkerInfo(resp string) (Worker) {
	worker := Worker{}
	msg := strings.Split(resp, " ")
	worker.Name = msg[0]
	worker.Status = "free"
	usage, _ := strconv.Atoi(msg[2])
	worker.Usage = usage
	worker.Tags = msg[3]
	port, _ := strconv.Atoi(msg[4])
	worker.Port = port
	jobsDone, _ := strconv.Atoi(msg[5])
	worker.JobsDone = jobsDone
	worker.Active = true
	worker.URL = "localhost:" + msg[4]
	return worker
}

func UpdateWorkerStatus(worker string, currStatus string){
	prevWorker := Workers[worker]
	newWorker := Worker{
		Name: prevWorker.Name,
		Tags: prevWorker.Tags,
		Status: currStatus,
		Usage: prevWorker.Usage,
		URL: prevWorker.URL,
		Active: prevWorker.Active,
		Port: prevWorker.Port,
		JobsDone: prevWorker.JobsDone,
	}
	Workers[worker] = newWorker
}

func Register(name string, num int) {
	actions[strconv.Itoa(num)] = Action{id: num, worker: name}
}

func GetWorker(id int) string {
	name := actions[strconv.Itoa(id)].worker
	return name
}

func GetWorkloadName(key string) (string) {
	return Workloads[key].Name
}

func UpdateStatus(name string) {
	if w, ok := Workers[name]; ok {
		if w.Status == "free" {
			w.Status = "busy"
		} else {
			w.Status = "free"
		}
	}
}

func UpdateUsage(name string) {
	if w, ok := Workers[name]; ok {
		w.Usage += 1
		w.JobsDone += 1
		Workers[name] = w
	}
}
