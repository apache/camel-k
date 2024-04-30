package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func main() {

	cmd := exec.Command("mvn", "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = fmt.Sprintf("Error: %v\n", err)
		return
	}

	for _, ln := range strings.Split(string(output), "\n") {
		switch {
		case strings.Contains(ln, "Apache Maven"):
			fmt.Printf("%v\n", ln)
			versionRegex := regexp.MustCompile(`Apache Maven ([1-9]+)(\.([0-9]+)){2,3}`)
			matches := versionRegex.FindStringSubmatch(ln)
			if len(matches) < 2 {
				_ = fmt.Sprintf("Unable to determine Apache Maven version: %s\n", ln)
				return
			}
		case strings.Contains(ln, "Java version"):
			fmt.Printf("%v\n", ln)
			versionRegex := regexp.MustCompile(`version: ([1-9]+)(\.([0-9]+)){2,3}`)
			matches := versionRegex.FindStringSubmatch(ln)
			if len(matches) < 2 {
				_ = fmt.Sprintf("Unable to determine Java version: %s\n", ln)
				return
			}
			majorVersion, err := strconv.Atoi(matches[1])
			if err != nil {
				_ = fmt.Sprintf("Error parsing Java version: %s - %v\n", ln, err)
				return
			}
			if majorVersion < 17 {
				_ = fmt.Sprintf("JDK version is below 17: %s\n", ln)
				return
			}
		}
	}
}
