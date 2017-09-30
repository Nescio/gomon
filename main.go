package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

var (
	command          string
	directory        string
	cmd              *exec.Cmd
	stdout           io.ReadCloser
	stderr           io.ReadCloser
	lastModification int64
	fileCount        int
)

func main() {
	if len(os.Args) < 2 {
		printRed("Please provide application (usage: gomon <path_to_application>)")
		return
	}
	command, _ = filepath.Abs(os.Args[1])
	directory = filepath.Dir(command)
	directory, _ = filepath.Abs(directory)

	printWhite("Monitoring: %s", directory)

	ticker := time.Tick(1 * time.Second)
	for range ticker {
		if checkFilesForChanges() {

			killApp()

			buildErr := buildApp()

			if buildErr == nil {
				launchApp()
			}
		}
	}
}

func checkFilesForChanges() bool {
	modified := false
	count := 0

	checkFunc := func(path string, info os.FileInfo, err error) error {
		ext := filepath.Ext(path)
		if ext == ".go" {
			f, err := os.Open(path)
			if err != nil {
				printRed("Error accessing file: %s", err)
			}
			stats, err := f.Stat()
			if err != nil {
				printRed("Error getting stats from opened file: %s", err)
			}

			modification := stats.ModTime().Unix()

			if modification > lastModification {
				lastModification = modification
				modified = true
			}
			count++
		}
		return err
	}

	err := filepath.Walk(directory, checkFunc)

	if err != nil {
		printRed("Error walking directory: %s", err)
	} else if count != fileCount {
		fileCount = count
		modified = true
	}
	return modified
}

func killApp() {
	if cmd != nil {
		err := cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			printRed("Error killing running process: %s", err)
		}
	}
}

func buildApp() error {
	printGreen("Rebuilding application...")
	cmd := exec.Command("go", "build")
	cmd.Dir = directory
	output, err := cmd.CombinedOutput()

	errn := cmd.Run()
	if errn != nil && errn.Error() != "exec: already started" {
		printRed("Error rebuilding application: %s", errn)
	}
	if err != nil {
		printRed("=== error ====\n\n%s\n==========", output)
	}
	return err
}

func launchApp() {
	cmd = exec.Command(command)
	cmd.Dir = directory

	stdout, _ = cmd.StdoutPipe()
	stderr, _ = cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		printRed("Error starting application: %s", err)
	}

	printGreen("Application started!")

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
}

func printRed(message string, args ...interface{}) {
	log.Printf("\033[1;31m"+message+"\033[0m\n", args...)
}
func printGreen(message string, args ...interface{}) {
	log.Printf("\033[1;32m"+message+"\033[0m\n", args...)
}
func printWhite(message string, args ...interface{}) {
	log.Printf("\033[1;37m"+message+"\033[0m\n", args...)
}
