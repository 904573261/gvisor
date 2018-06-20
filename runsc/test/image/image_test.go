// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package image provides end-to-end image tests for runsc. These tests require
// docker and runsc to be installed on the machine.
//
// The tests expect the runtime name to be provided in the RUNSC_RUNTIME
// environment variable (default: runsc-test).
//
// Each test calls docker commands to start up a container, and tests that it is
// behaving properly, like connecting to a port or looking at the output. The
// container is killed and deleted at the end.
package image

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"gvisor.googlesource.com/gvisor/runsc/test/testutil"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func runtime() string {
	r := os.Getenv("RUNSC_RUNTIME")
	if r == "" {
		return "runsc-test"
	}
	return r
}

func mountArg(source, target string) string {
	return fmt.Sprintf("%s:%s", source, target)
}

func getLocalPath(file string) string {
	return path.Join(".", file)
}

type docker struct {
	runtime string
	name    string
}

func makeDocker(namePrefix string) docker {
	suffix := fmt.Sprintf("-%06d", rand.Int())[:7]
	return docker{name: namePrefix + suffix, runtime: runtime()}
}

// do executes docker command.
func (d *docker) do(args ...string) (string, error) {
	cmd := exec.Command("docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error executing docker %s: %v", args, err)
	}
	return string(out), nil
}

// run calls 'docker run' with the arguments provided.
func (d *docker) run(args ...string) (string, error) {
	a := []string{"run", "--runtime", d.runtime, "--name", d.name, "-d"}
	a = append(a, args...)
	return d.do(a...)
}

// cleanUp kills and deletes the container.
func (d *docker) cleanUp() error {
	if _, err := d.do("kill", d.name); err != nil {
		return fmt.Errorf("error killing container %q: %v", d.name, err)
	}
	if _, err := d.do("rm", d.name); err != nil {
		return fmt.Errorf("error deleting container %q: %v", d.name, err)
	}
	return nil
}

// findPort returns the host port that is mapped to 'sandboxPort'. This calls
// docker to allocate a free port in the host and prevent conflicts.
func (d *docker) findPort(sandboxPort int) (int, error) {
	format := fmt.Sprintf(`{{ (index (index .NetworkSettings.Ports "%d/tcp") 0).HostPort }}`, sandboxPort)
	out, err := d.do("inspect", "-f", format, d.name)
	if err != nil {
		return -1, fmt.Errorf("error retrieving port: %v", err)
	}
	port, err := strconv.Atoi(strings.TrimSuffix(string(out), "\n"))
	if err != nil {
		return -1, fmt.Errorf("error parsing port %q: %v", out, err)
	}
	return port, nil
}

// waitForOutput calls 'docker logs' to retrieve containers output and searches
// for the given pattern.
func (d *docker) waitForOutput(pattern string, timeout time.Duration) error {
	re := regexp.MustCompile(pattern)
	for exp := time.Now().Add(timeout); time.Now().Before(exp); {
		out, err := d.do("logs", d.name)
		if err != nil {
			return err
		}
		if re.MatchString(out) {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for output %q", re.String())
}

func TestHelloWorld(t *testing.T) {
	d := makeDocker("hello-test")
	if out, err := d.run("hello-world"); err != nil {
		t.Fatalf("docker run failed: %v\nout: %s", err, out)
	}
	defer d.cleanUp()

	if err := d.waitForOutput(".*Hello from Docker!.*", 5*time.Second); err != nil {
		t.Fatalf("docker didn't say hello: %v", err)
	}
}

func TestHttpd(t *testing.T) {
	d := makeDocker("http-test")

	// Create temp directory to copy htdocs files. The sandbox doesn't have access
	// to files in the test dir.
	dir, err := ioutil.TempDir("", "httpd")
	if err != nil {
		t.Fatalf("ioutil.TempDir failed: %v", err)
	}
	if err := os.Chmod(dir, 0777); err != nil {
		t.Fatalf("os.Chmod(%q, 0777) failed: %v", dir, err)
	}
	src := getLocalPath("latin10k.txt")
	dst := path.Join(dir, "latin10k.txt")
	if err := testutil.Copy(src, dst); err != nil {
		t.Fatalf("testutil.Copy(%q, %q) failed: %v", src, dst, err)
	}

	// Start the container.
	if out, err := d.run("-p", "80", "-v", mountArg(dir, "/usr/local/apache2/htdocs"), "httpd"); err != nil {
		t.Fatalf("docker run failed: %v\nout: %s", err, out)
	}
	defer d.cleanUp()

	// Find where port 80 is mapped to.
	port, err := d.findPort(80)
	if err != nil {
		t.Fatalf("docker.findPort(80) failed: %v", err)
	}

	// Wait until it's up and running.
	if err := d.waitForOutput(".*'httpd -D FOREGROUND'.*", 5*time.Second); err != nil {
		t.Fatalf("docker.WaitForOutput() timeout: %v", err)
	}

	url := fmt.Sprintf("http://localhost:%d/not-found", port)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("error reaching http server: %v", err)
	}
	if want := http.StatusNotFound; resp.StatusCode != want {
		t.Errorf("Wrong response code, got: %d, want: %d", resp.StatusCode, want)
	}

	url = fmt.Sprintf("http://localhost:%d/latin10k.txt", port)
	resp, err = http.Get(url)
	if err != nil {
		t.Fatalf("Error reaching http server: %v", err)
	}
	if want := http.StatusOK; resp.StatusCode != want {
		t.Errorf("Wrong response code, got: %d, want: %d", resp.StatusCode, want)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading http response: %v", err)
	}
	defer resp.Body.Close()

	// READALL is the last word in the file. Ensures everything was read.
	if want := "READALL"; strings.HasSuffix(string(body), want) {
		t.Errorf("response doesn't contain %q, resp: %q", want, body)
	}
}

func MainTest(m *testing.M) {
	// Check correct docker is installed.
	cmd := exec.Command("docker", "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Error running %q: %v", "docker version", err)
	}
	re := regexp.MustCompile(`Version:\s+(\d+)\.(\d+)\.\d.*`)
	matches := re.FindStringSubmatch(string(out))
	if len(matches) != 3 {
		log.Fatalf("Invalid docker output: %s", out)
	}
	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	if major < 17 || (major == 17 && minor < 9) {
		log.Fatalf("Docker version 17.09.0 or greater is required, found: %02d.%02d", major, minor)
	}

	os.Exit(m.Run())
}
