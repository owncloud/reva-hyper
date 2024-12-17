package ocis

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"ociswrapper/common"
	"ociswrapper/log"
	"ociswrapper/ocis/config"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

var cmd *exec.Cmd
var retryCount = 0
var stopSignal = false
var EnvConfigs = []string{}
var runningCommands = make(map[string]int) // Maps unique IDs to PIDs

func Start(envMap []string) {
	StartService("", envMap)
}

func Stop() (bool, string) {
	log.Println(fmt.Sprintf("Stop ocis check cmd %s\n", cmd))
	log.Println("Stopping oCIS server...")
	stopSignal = true

	for listservice, pid := range runningCommands {
	 	log.Println(fmt.Sprintf("Services running before terminating: %s with process and id: %v\n", listservice, pid))
	 	process, err := os.FindProcess(pid)
	 	err = process.Signal(syscall.SIGINT)
        	if err != nil {
        		if !strings.HasSuffix(err.Error(), "process already finished") {
        			log.Fatalln(err)
        		} else {
        			return true, "oCIS server is already stopped"
        		}
        	}
        	process.Wait()
	}

	cmd = nil
	success, message := waitUntilCompleteShutdown()
	return success, message
}

func Restart(envMap []string) (bool, string) {
	log.Println(fmt.Sprintf("Restarting ocis check cmd %s\n", cmd))
	log.Println(fmt.Sprintf("Restaring ocis with rollback os environ %s\n", envMap))
	log.Println(fmt.Sprintf("OS environment: %s\n", os.Environ()))
	go Stop()

	log.Println("Restarting oCIS server...")
	common.Wg.Add(1)
	go Start(envMap)

	return WaitForConnection()
}

func IsOcisRunning() bool {
	if cmd != nil {
		return cmd.Process.Pid > 0
	}
	return false
}

func waitAllServices(startTime time.Time, timeout time.Duration) {
	timeoutS := timeout * time.Second

	c := exec.Command(config.Get("bin"), "list")
	_, err := c.CombinedOutput()
	if err != nil {
		if time.Since(startTime) <= timeoutS {
			time.Sleep(500 * time.Millisecond)
			waitAllServices(startTime, timeout)
		}
		return
	}
	log.Println("All services are up")
}

func WaitForConnection() (bool, string) {
	waitAllServices(time.Now(), 30)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	// 30 seconds timeout
	timeoutValue := 30 * time.Second

	client := http.Client{
		Timeout:   timeoutValue,
		Transport: transport,
	}

	var req *http.Request
	if config.Get("adminUsername") != "" && config.Get("adminPassword") != "" {
		req, _ = http.NewRequest("GET", config.Get("url")+"/graph/v1.0/me/drives", nil)
		req.SetBasicAuth(config.Get("adminUsername"), config.Get("adminPassword"))
	} else {
		req, _ = http.NewRequest("GET", config.Get("url")+"/ocs/v1.php/cloud/capabilities?format=json", nil)
	}

	timeout := time.After(timeoutValue)

	for {
		select {
		case <-timeout:
			log.Println(fmt.Sprintf("%v seconds timeout waiting for oCIS server", int64(timeoutValue.Seconds())))
			return false, "Timeout waiting for oCIS server to start"
		default:
			req.Header.Set("X-Request-ID", "ociswrapper-"+strconv.Itoa(int(time.Now().UnixMilli())))

			res, err := client.Do(req)
			if err != nil || res.StatusCode != 200 {
				// 500 milliseconds poll interval
				time.Sleep(500 * time.Millisecond)
				continue
			}

			log.Println("oCIS server is ready to accept requests")
			return true, "oCIS server is up and running"
		}
	}
}

func waitUntilCompleteShutdown() (bool, string) {
	log.Println("Process found. Waiting... waitUntilCompleteShutdown")
	timeout := 30 * time.Second
	startTime := time.Now()

	c := exec.Command("sh", "-c", "ps ax | grep 'ocis server' | grep -v grep | awk '{print $1}'")
	output, err := c.CombinedOutput()
	if err != nil {
		log.Println(err.Error())
	}
	for strings.TrimSpace(string(output)) != "" {
		output, _ = c.CombinedOutput()
		log.Println("Process found. Waiting...")

		if time.Since(startTime) >= timeout {
			log.Println(fmt.Sprintf("Unable to kill oCIS server after %v seconds", int64(timeout.Seconds())))
			return false, "Timeout waiting for oCIS server to stop"
		}
	}
	return true, "oCIS server stopped successfully"
}

func RunCommand(command string, inputs []string) (int, string) {
	logs := new(strings.Builder)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// build the command
	cmdArgs := strings.Split(command, " ")
	c := exec.CommandContext(ctx, config.Get("bin"), cmdArgs...)

	// Start the command with a pty (pseudo terminal)
	// This is required to interact with the command
	ptyF, err := pty.Start(c)
	if err != nil {
		log.Panic(err)
	}
	defer ptyF.Close()

	for _, input := range inputs {
		fmt.Fprintf(ptyF, "%s\n", input)
	}

	var cmdOutput string
	if err := c.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			cmdOutput = "Command timed out:\n"
		}
	}

	// Copy the logs from the pty
	io.Copy(logs, ptyF)
	cmdOutput += logs.String()

	// TODO: find if there is a better way to remove stdins from the output
	cmdOutput = strings.TrimLeft(cmdOutput, strings.Join(inputs, "\r\n"))

	return c.ProcessState.ExitCode(), cmdOutput
}

func RunOcisService(service string, envMap []string) {
	log.Println(fmt.Sprintf("Environment variable envMap: %s\n", envMap))
	StartService(service, envMap)
}

// startService is a common function for starting a service (ocis or other)
func StartService(service string, envMap []string) {
	log.Println(fmt.Sprintf("Start service: %s with Environment variable envMap: %s\n", service, envMap))
	// Initialize command args based on service presence
	cmdArgs := []string{"server"} // Default command args

	if service != "" {
		// Directly append service if provided
		cmdArgs = append([]string{service}, cmdArgs...)
	}

	// wait for the log scanner to finish
	var wg sync.WaitGroup
	wg.Add(2)

	stopSignal = false
	if retryCount == 0 {
		defer common.Wg.Done()
	}

	// Initialize the command
	cmd = exec.Command(config.Get("bin"), cmdArgs...)

	// Use the provided envMap if not empty, otherwise use EnvConfigs
	if len(envMap) == 0 {
		cmd.Env = append(os.Environ(), EnvConfigs...)
		log.Println(fmt.Sprintf("OS environment variables while running ocis service: %s\n", cmd.Env))
		log.Println(fmt.Sprintf("OS environment: %s\n", os.Environ()))
	} else {
		cmd.Env = append(os.Environ(), envMap...)
		log.Println(fmt.Sprintf("OS environment variables while running ocis service: %s\n", cmd.Env))
		log.Println(fmt.Sprintf("OS environment: %s\n", os.Environ()))
	}

	logs, err := cmd.StderrPipe()
	if err != nil {
		log.Panic(err)
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		log.Panic(err)
	}

// 	log.Println(fmt.Sprintf("command env used to start service %s\n", cmd.Env))
// 	log.Println(fmt.Sprintf("command used to start service %s\n", cmd))

	err = cmd.Start()

	if err != nil {
		log.Panic(err)
	}

	logScanner := bufio.NewScanner(logs)
	outputScanner := bufio.NewScanner(output)
	outChan := make(chan string)

	// If service is an empty string, set the PID for "ocis"
	if service == "" {
		runningCommands["ocis"] = cmd.Process.Pid
	} else {
		runningCommands[service] = cmd.Process.Pid
	}

	log.Println("Started oCIS processes:")
	for listservice, pid := range runningCommands {
	 	log.Println(fmt.Sprintf("Service started: %s with process and id: %v\n", listservice, pid))

	}

	// Read the logs when the 'ocis server' command is running
	go func() {
		defer wg.Done()
		for logScanner.Scan() {
			outChan <- logScanner.Text()
		}
	}()

	go func() {
		defer wg.Done()
		for outputScanner.Scan() {
			outChan <- outputScanner.Text()
		}
	}()

	// Fetch logs from the channel and print them
	go func() {
		for s := range outChan {
			fmt.Println(s)
		}
	}()

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			status := exitErr.Sys().(syscall.WaitStatus)
			// retry only if oCIS server exited with code > 0
			// -1 exit code means that the process was killed by a signal (syscall.SIGINT)
			if status.ExitStatus() > 0 && !stopSignal {
				waitUntilCompleteShutdown()

				log.Println(fmt.Sprintf("oCIS server exited with code %v", status.ExitStatus()))

				// retry to start oCIS server
				retryCount++
				maxRetry, _ := strconv.Atoi(config.Get("retry"))
				if retryCount <= maxRetry {
					wg.Wait()
					close(outChan)
					log.Println(fmt.Sprintf("Retry starting oCIS server... (retry %v)", retryCount))
					// wait 500 milliseconds before retrying
					time.Sleep(500 * time.Millisecond)
					StartService(service, envMap)
					return
				}
			}
		}
	}

	log.Println(fmt.Sprintf(" ---- ocis start service ending line---- %s\n", cmd))
	wg.Wait()
	close(outChan)
}

// Stop oCIS service or a specific service by its unique identifier
func StopService(service string) (bool, string) {
	for listservice, pid := range runningCommands {
	 	log.Println(fmt.Sprintf("Services running before terminating: %s with process and id: %v\n", listservice, pid))
	}

	pid, exists := runningCommands[service]
	log.Println(fmt.Sprintf("Services process id to terminate: %s\n", pid))
	if !exists {
		return false, fmt.Sprintf("Service %s is not running", service)
	}

	// Find the process by PID and send SIGINT to stop it
	process, err := os.FindProcess(pid)
	log.Println(fmt.Sprintf("Found process to terminate in os: %s\n", pid))

	if err != nil {
		log.Println(fmt.Sprintf("Failed to find process: %v", err))
		return false, fmt.Sprintf("Failed to find process with ID %d", pid)
	}

	err = process.Signal(syscall.SIGINT)
	log.Println("Process terminated using signal")
	if err != nil {
		log.Println(fmt.Sprintf("Failed to send signal: %v", err))
		return false, fmt.Sprintf("Failed to stop service with PID %d", pid)
	}
    time.Sleep(30 * time.Second)
	process.Wait()
	log.Println("Process terminating process.wait")
	delete(runningCommands, service)
	for listservice, pid := range runningCommands {
	 	log.Println(fmt.Sprintf("Service list after deleteing %s service. list contain service: %s with process and id: %v\n", service, listservice, pid))
	}
	return true, fmt.Sprintf("Service %s stopped successfully", service)
}
