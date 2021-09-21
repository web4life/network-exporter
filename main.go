package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func execute() map[string]float64 {

	cmd := "netstat -nat | grep -i 'established' |  awk '{print $5}' | cut -d: -f1 | sed -e '/^$/d' | sed -e '/[A-Za-z]/d' | sort | uniq -c | sort -nr"
	out, err := exec.Command("bash", "-c", cmd).Output()

	if err != nil {
		fmt.Printf("%s", err)
	}

	output := string(out[:])
	scanner := bufio.NewScanner(strings.NewReader(output))
	ipsMap := make(map[string]float64)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		ipsSlice := strings.Split(line, " ")
		count, err := strconv.ParseFloat(ipsSlice[0], 64)
		ip := ipsSlice[1]

		if err != nil {
			fmt.Printf("%s", err)
		}

		ipsMap[ip] = count
	}
	if scanner.Err() != nil {
		log.Println(scanner.Err())
	}
	return ipsMap
}

func recordMetrics() {
	go func() {
		for {
			output := execute()
			for ip, count := range output {
				netstatGauge.With(prometheus.Labels{"ip": ip}).Set(count)
			}
			time.Sleep(15 * time.Second)
			netstatGauge.Reset()
		}
	}()
}

var (
	netstatGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "netstat_number_of_connections",
		Help: "Number of tcp connections to server by ips",
	},
		[]string{"ip"})
)

func main() {
	if err := prometheus.Register(netstatGauge); err != nil {
		fmt.Println(err)
	}

	recordMetrics()
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
