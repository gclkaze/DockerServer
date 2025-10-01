package dockerserver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/gclkaze/DockerServer/dockerserver/stdlogger"
	"github.com/magiconair/properties"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type DockerServer struct {
	onError             bool
	properties          *properties.Properties
	applicationFileName string
	workingDirectory    string
	jobsPrefix          string
	logger              *stdlogger.STDLogger
	ready               *AtomicBool
	port                int
	host                string
	jobs                cmap.ConcurrentMap[string, []*DockerJob]
	listeners           cmap.ConcurrentMap[string, net.Listener]
	socketMutex         *sync.Mutex
}

var TheServer *DockerServer

func InitDockerServer() {
	server := NewDockerServer("")
	server.Load()

	TheServer = server
}

func GetServerInstance() *DockerServer {
	return TheServer
}

func NewDockerServer(props string) *DockerServer {
	appFile := ""
	if appFile == "" {
		appFile = "dockerserver\\resources\\application.properties"
	}
	return &DockerServer{onError: false, applicationFileName: appFile, logger: stdlogger.NewSTDLogger()}
}

func (server DockerServer) GetPort() int {
	return server.port
}

func (server DockerServer) GetHost() string {
	return server.host
}

func (server DockerServer) GetAddress() string {
	return server.host + ":" + strconv.Itoa(server.port)
}

func (server *DockerServer) Load() bool {
	server.properties = properties.MustLoadFile(server.applicationFileName, properties.UTF8)

	server.workingDirectory = server.properties.GetString("workingDirectory", "")
	if server.workingDirectory == "" || !server.resourceExists(server.workingDirectory) {
		server.logger.PrintErrorMessage("docker server", fmt.Sprintf("working directory %s does not exist", server.workingDirectory))
		return false
	}

	server.jobsPrefix = server.properties.GetString("jobsPrefix", "")
	jobs, err := server.GetCurrentDockerSessions()

	if err != nil {
		server.logger.PrintErrorMessage("docker server", err.Error())
		return false
	}
	server.port = server.properties.GetInt("port", 9977)
	server.host = server.properties.GetString("host", "localhost")

	if len(jobs) > 0 {
		server.cleanOrphanJobs(jobs)
	}

	server.jobs = cmap.New[[]*DockerJob]()
	server.listeners = cmap.New[net.Listener]()

	server.socketMutex = &sync.Mutex{}
	server.ready = &AtomicBool{}
	server.ready.CompareAndSwap(false, true)
	return true
}

func (server *DockerServer) GetCurrentDockerSessions() (jobs []DockerJob, err error) {
	res, er := server.executeCommand("docker ps --format json")
	if er != nil {
		return nil, er
	}
	all, err := server.extractDockerJobsFromString(res)
	if err != nil {
		return nil, err
	}
	all = server.filterEvaSessions(all, func(u DockerJob) bool {
		return strings.HasPrefix(u.Names, server.jobsPrefix)
	})
	return all, err
}

func (server DockerServer) filterEvaSessions(jobs []DockerJob, test func(DockerJob) bool) []DockerJob {
	var result []DockerJob
	for _, u := range jobs {
		if test(u) {
			result = append(result, u)
		}
	}
	return result
}

func (server DockerServer) extractDockerJobsFromString(res string) ([]DockerJob, error) {
	if res[0] != '[' {
		res = "[" + res + "]"
	}

	if strings.Contains(res, "}\n{") {
		res = strings.ReplaceAll(res, "}\n{", "}\n,{")
	}
	fmt.Print(res)
	var jobs []DockerJob
	if err := json.Unmarshal([]byte(res), &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (server DockerServer) buildAndRunDockerSession(name string) error {
	if strings.Contains(name, " ") {
		name = strings.ReplaceAll(name, " ", "-")
	}
	imgName := name + "-image"
	res, err := server.executeCommand("docker build -t " + imgName + " dockerserver")
	if err != nil {
		server.logger.PrintErrorMessage("docker server", res)
		server.logger.PrintErrorMessage("docker server", err.Error())
		return err
	}

	res, err = server.executeCommand("docker run -it --rm -d --name " + name + " " + imgName)
	if err != nil {
		server.logger.PrintErrorMessage("docker server", res)
		server.logger.PrintErrorMessage("docker server", err.Error())
	}
	return err
}

func (server DockerServer) spawnDockerSession(_ string, programId string) (*DockerJob, error) {
	name := server.jobsPrefix + programId
	//res, err := server.executeCommand("docker run -d --name " + name + " ubuntu:22.04 sleep infinity")
	err := server.buildAndRunDockerSession(name)
	if err != nil {
		return nil, err
	}
	jobs, err := server.GetCurrentDockerSessions()
	if err != nil {
		return nil, err
	}
	job := server.filterEvaSessions(jobs, func(u DockerJob) bool {
		return strings.HasPrefix(u.Names, name)
	})

	if len(job) == 0 {
		//server.logger.PrintErrorMessage("docker server", res)
		server.logger.PrintErrorMessage("docker server", "no container with name "+name)
		return nil, fmt.Errorf("Couldn't find container with name " + name)
	}

	if len(job) > 1 {
		//server.logger.PrintErrorMessage("docker server", res)
		server.logger.PrintErrorMessage("docker server", "more than 1 containers with name "+name)
		return nil, fmt.Errorf("More than 1 containers with name " + name)
	}
	return &job[0], nil
}

func (server *DockerServer) CreateListener() (net.Listener, error) {
	server.socketMutex.Lock()
	listener, err := net.Listen("tcp", ":0")
	server.socketMutex.Unlock()

	return listener, err
}

func (server *DockerServer) CreateSession(config string, programId string) (string, error) {
	jobList, ok := server.jobs.Get(programId)
	name := programId
	if ok {
		sz := len(jobList)
		name = programId + "-" + strconv.Itoa(sz)
	}
	job, err := server.spawnDockerSession(config, name)
	if err != nil {
		return "", err
	}

	//lets create a socket for the session
	listener, err := server.CreateListener()
	if err != nil {
		server.logger.PrintErrorMessage("docker server", fmt.Sprintf("Failed to listen: %v", err))
		return "", nil
	}
	// Print the actual port assigned

	fmt.Printf("Server listening on %s\n", listener.Addr().String())
	socketNumber := listener.Addr().String()
	job.SocketNumber = socketNumber
	jobList = append(jobList, job)
	server.jobs.Set(programId, jobList)

	server.listeners.Set(socketNumber, listener)

	return socketNumber, nil
}

func (server DockerServer) findJobAssociatedWithSocket(socketnumber string) *DockerJob {
	for item := range server.jobs.IterBuffered() {
		l := item.Val
		for i := 0; i < len(l); i++ {
			if l[i].SocketNumber == socketnumber {
				return l[i]
			}
		}
	}
	return nil
}

func (server DockerServer) findJobAssociatedWithName(name string) *DockerJob {
	for item := range server.jobs.IterBuffered() {
		l := item.Val
		for i := 0; i < len(l); i++ {
			if l[i].Names == name {
				return l[i]
			}
		}
	}
	return nil
}

func (server *DockerServer) OpenSession(socketnumber string, config string, programId string) {
	listener, ok := server.listeners.Get(socketnumber)
	if !ok {
		server.logger.PrintErrorMessage("docker server", fmt.Sprintf("Failed to find socket: %s", socketnumber))
		return
	}

	//lets find also the terminal
	wsp := server.findJobAssociatedWithSocket(socketnumber)
	if wsp == nil {
		server.logger.PrintErrorMessage("docker server", fmt.Sprintf("Failed to find Workspace for socket: %s", socketnumber))
		return
	}
	go func() {

		for {
			conn, err := listener.Accept()
			if err != nil {
				server.logger.PrintErrorMessage("docker server", fmt.Sprintf("Failed to listen: %v", err))
				continue
			}

			go server.handleConnection(conn, wsp, config, programId)
		}
	}()
}

func (server *DockerServer) cleanSocket(conn net.Conn, wsp *DockerJob, config string, programId string) {
	conn.Close()
	server.listeners.Remove(wsp.SocketNumber)
	server.StopSession(config, programId)
}

func (server *DockerServer) handleConnection(conn net.Conn, wsp *DockerJob, config string, programId string) {
	for {
		cmd, err := server.readCommand(conn)
		if err != nil {
			server.logger.PrintErrorMessage("docker server", fmt.Sprintf("Failed to read message: %v", err))
			server.cleanSocket(conn, wsp, config, programId)
			return
		}

		if strings.HasPrefix(cmd, "exit") {
			server.cleanSocket(conn, wsp, config, programId)
			return
		}
		theCmd := "docker exec " + wsp.Names + ` bash -c`
		//server.logger.PrintMessage("docker server", "executing cmd "+theCmd)
		server.executeCommandRedirectionToSocket(theCmd, cmd, conn)
	}
}

func (server *DockerServer) readCommand(conn net.Conn) (cmd string, err error) {
	reader := bufio.NewReader(conn)
	var builder strings.Builder

	for {
		// Read until newline or buffer fill
		chunk, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading:", err)
			return "", err
		}

		builder.WriteString(chunk)

		// Check if message ends with "eof"
		if strings.Contains(builder.String(), "\n") {
			message := strings.TrimSuffix(builder.String(), "\n")
			fmt.Printf("Received complete message: %q\n", message)
			return message, nil
		}
	}
}

func (server *DockerServer) cleanOrphanJobs(jobs []DockerJob) (bool, error) {
	server.logger.PrintMessage("docker server", "Cleaning "+strconv.Itoa(len(jobs))+" orphan jobs.")

	for i := 0; i < len(jobs); i++ {
		server.logger.PrintMessage("docker server", strconv.Itoa(i)+". Cleaning "+jobs[i].Names+" docker job.")
		_, err := server.EliminateContainer(jobs[i].Names)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func (server *DockerServer) WriteFile(content string, programId string, filename string) (string, error) {
	name := server.jobsPrefix + programId
	job := server.findJobAssociatedWithName(name)
	if job == nil {
		return "", fmt.Errorf("No such docker job " + name)
	}
	theCmd := "docker exec " + job.Names + ` bash -c`
	res, err := server.executeVectorCommand(theCmd, "echo \""+content+"\" > "+filename)
	return res, err
}

func (server *DockerServer) EliminateContainer(name string) (bool, error) {
	_, err := server.executeCommand("docker stop " + name)
	if err != nil {
		server.logger.PrintErrorMessage("docker server", err.Error())
		return false, err
	}

	_, err = server.executeCommand("docker rmi " + name + "-image")
	if err != nil {
		server.logger.PrintErrorMessage("docker server", err.Error())
		return true, nil //err
	}

	_, err = server.executeCommand("docker rm " + name)
	if err != nil {
		server.logger.PrintErrorMessage("docker server", err.Error())
		return true, nil //err
	}
	return true, nil
}

func (server *DockerServer) StopSession(config string, programId string) (bool, error) {
	jobList, ok := server.jobs.Get(programId)
	if !ok {
		return true, nil
	}

	for i := 0; i < len(jobList); i++ {
		_, err := server.EliminateContainer(jobList[i].Names)
		if err != nil {
			return false, err
		}
	}

	server.jobs.Remove(programId)
	return true, nil
}

func (server DockerServer) executeCmd(cmdStr string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/C", cmdStr)
	} else if runtime.GOOS == "linux" {
		return exec.Command("sh", "-c", cmdStr)
	}
	return nil
}

func (server DockerServer) executeCmdWithCmdArg(cmdStr []string, main string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		finalArray := []string{"/C"}
		finalArray = append(finalArray, cmdStr...)
		finalArray = append(finalArray, main)
		cmd := exec.Command("cmd", finalArray...)
		return cmd
	} else if runtime.GOOS == "linux" {
		finalArray := []string{"-c"}
		finalArray = append(finalArray, cmdStr...)
		finalArray = append(finalArray, main)
		cmd := exec.Command("sh", finalArray...)
		return cmd
	}
	return nil
}

func (server DockerServer) executeCommand(cmdStr string) (res string, er error) {
	cmd := server.executeCmd(cmdStr)
	if cmd == nil {
		return "", fmt.Errorf("couldn't execute Command %s ", cmdStr)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), err
}

func (server DockerServer) executeCommandRedirectionToSocket(cmdStr string, theCmd string, conn net.Conn) (res string, er error) {
	endMessage := "<!--END_OF_MESSAGE-->"
	cmd := server.executeCmdWithCmdArg(strings.Split(cmdStr, " "), theCmd)

	cmd.Stdout = conn
	cmd.Stderr = conn

	exitStatus := 0

	if err := cmd.Run(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			exitStatus = exiterr.ExitCode()
		}
		server.logger.PrintErrorMessage("docker server", fmt.Sprintf("Command failed: %v", err))
		endMessage = "!!!-" + strconv.Itoa(exitStatus) + "-!!!" + endMessage
		conn.Write([]byte(endMessage))
		return "", err
	}
	endMessage = "!!!-" + strconv.Itoa(exitStatus) + "-!!!" + endMessage
	conn.Write([]byte(endMessage))
	return "", nil
}

func (server DockerServer) executeVectorCommand(cmdStr string, theCmd string) (res string, er error) {
	cmd := server.executeCmdWithCmdArg(strings.Split(cmdStr, " "), theCmd)
	if cmd == nil {
		return "", fmt.Errorf("couldn't execute Command %s ", cmdStr)
	}

	// Run the command and check for error
	if err := cmd.Run(); err != nil {
		return "", err
	}

	return "", nil

}

func (server DockerServer) resourceExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}
