package ocis

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"ociswrapper/common"
	"ociswrapper/log"
	"ociswrapper/ocis/config"

	"github.com/creack/pty"
)

var cmd *exec.Cmd
var retryCount = 0
var stopSignal = false
var EnvConfigs = []string{}

func Start(envMap []string) {
	// wait for the log scanner to finish
	var wg sync.WaitGroup
	wg.Add(2)

	stopSignal = false
	if retryCount == 0 {
		defer common.Wg.Done()
	}

	cmd = exec.Command(config.Get("bin"), "server")
	if envMap == nil {
		cmd.Env = append(os.Environ(), EnvConfigs...)
	} else {
		cmd.Env = append(os.Environ(), envMap...)
	}

	logs, err := cmd.StderrPipe()
	if err != nil {
		log.Panic(err)
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		log.Panic(err)
	}

	err = cmd.Start()
	if err != nil {
		log.Panic(err)
	}

	logScanner := bufio.NewScanner(logs)
	outputScanner := bufio.NewScanner(output)
	outChan := make(chan string)

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
					Start(envMap)
					return
				}
			}
		}
	}
	wg.Wait()
	close(outChan)
}

func Stop() (bool, string) {
	log.Println("Stopping oCIS server...")
	stopSignal = true

	if cmd == nil {
		return true, "oCIS server is not running"
	}

	err := cmd.Process.Signal(syscall.SIGINT)
	if err != nil {
		if !strings.HasSuffix(err.Error(), "process already finished") {
			log.Fatalln(err)
		} else {
			return true, "oCIS server is already stopped"
		}
	}
	cmd.Process.Wait()
	success, message := waitUntilCompleteShutdown()

	cmd = nil
	return success, message
}

func Restart(envMap []string) (bool, string) {
	Stop()

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

func runOcisService(service string) {
	// wait for the log scanner to finish
	var wg sync.WaitGroup
	wg.Add(2)

	stopSignal = false
	if retryCount == 0 {
		defer common.Wg.Done()
	}

	cmd = exec.Command(config.Get("bin"), service , "server")
// 		output, err := cmd.CombinedOutput()
//     	if err != nil {
//     		fmt.Printf("Error running service %s: %v\nOutput: %s\n", service, err, output)
//     		return
//     	}
//     	fmt.Printf("Service %s started successfully: %s\n", service, output)

	logs, err := cmd.StderrPipe()
	if err != nil {
		log.Panic(err)
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		log.Panic(err)
	}

	err = cmd.Start()
	if err != nil {
		log.Panic(err)
	}

	logScanner := bufio.NewScanner(logs)
	outputScanner := bufio.NewScanner(output)
	outChan := make(chan string)

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

				log.Println(fmt.Sprintf("oCIS service server exited with code %v", status.ExitStatus()))

				// retry to start oCIS server
				retryCount++
				maxRetry, _ := strconv.Atoi(config.Get("retry"))
				if retryCount <= maxRetry {
					wg.Wait()
					close(outChan)
					log.Println(fmt.Sprintf("Retry starting oCIS server... (retry %v)", retryCount))
					// wait 500 milliseconds before retrying
					time.Sleep(500 * time.Millisecond)
					runOcisService(service)
					return
				}
			}
		}
	}
	wg.Wait()
	close(outChan)
}
