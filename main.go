package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/pterm/pterm"
)

type ipResult struct {
	Ip       net.IP
	HttpCode string
}

func main() {
	var chunksCount int
	var timeout int

	flag.IntVar(&chunksCount, "chunks", 50000, "Number of simultaneous requests")
	flag.IntVar(&timeout, "timeout", 2, "Timeout in seconds")
	flag.Parse()

	ipRanges := []string{
		"173.245.48.0/20",
		"103.21.244.0/22",
		"103.22.200.0/22",
		"103.31.4.0/22",
		"141.101.64.0/18",
		"108.162.192.0/18",
		"190.93.240.0/20",
		"188.114.96.0/20",
		"197.234.240.0/22",
		"198.41.128.0/17",
		"162.158.0.0/15",
		"104.16.0.0/13",
		"104.24.0.0/14",
		"172.64.0.0/13",
		"131.0.72.0/22",
	}

	var wg sync.WaitGroup
	sem := make(chan bool, chunksCount)

	var mutex = &sync.Mutex{}

	var ipResults []ipResult

	pterm.Println()

	for _, cidr := range ipRanges {
		ip, ipnet, _ := net.ParseCIDR(cidr)

		for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
			sem <- true
			wg.Add(1)

			go func(ip net.IP) {
				defer func() {
					<-sem
					wg.Done()
				}()

				// Use the `curl` command to check if the IP address responds
				out, _ := exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "-m", fmt.Sprintf("%d", timeout), fmt.Sprintf("http://%s", ip)).Output()

				httpCode := string(out)
				addIp := false

				if httpCode == "000" {
					pterm.Error.Printf("%s -> Timed Out!\n", ip)
				} else if httpCode == "200" {
					pterm.Success.Printf("%s -> %s\n", ip, httpCode)

					addIp = true
				} else {
					pterm.Warning.Printf("%s -> %s\n", ip, httpCode)

					addIp = true
				}

				if addIp {
					mutex.Lock()

					ipResults = append(ipResults, ipResult{
						Ip:       ip,
						HttpCode: httpCode,
					})

					mutex.Unlock()
				}
			}(ip)
		}
	}

	wg.Wait()

	writeToFile(ipResults)

	pterm.Println()
	pterm.Println()
	pterm.Success.Printf("All Done!")
}

// Helper function to increment the IP address
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++

		if ip[j] > 0 {
			break
		}
	}
}

func writeToFile(ipResults []ipResult) {
	configDir := filepath.Join(os.Getenv("HOME"), "Desktop")

	jsonData, err := json.MarshalIndent(ipResults, "", "    ")

	if err != nil {
		panic(err)
	}

	file := filepath.Join(configDir, "ip-result.json")

	err = os.WriteFile(file, jsonData, 0644)

	if err != nil {
		panic(err)
	}
}
