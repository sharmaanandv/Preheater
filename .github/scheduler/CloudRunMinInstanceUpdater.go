package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

const ServiceName string = "{service_name}"
const Token = "TOKEN"
const TRAFFIC = "TRAFFIC"
const LowTraffic = "low"
const HighTraffic = "high"
const REGION = "REGION"
const IntProjectId = "ca-kijiji-int-t0z7"

var servicefile = "scheduler/SchedulerServices.csv"

type Service struct {
	name         string
	minInstances int
}

func main() {
	// Uncomment below for local debugging
	//os.Setenv(Token, "my-token")
	//os.Setenv(TRAFFIC, "high")
	//os.Setenv(REGION, "us-central1")
	//servicefile = ".github/scheduler/SchedulerServices.csv"

	servicefile = os.Getenv("CSVFILE")
	fmt.Println("Updating Min Instances...")
	validateEnv()
	filteredServices := getWhiteListedServices(os.Getenv(TRAFFIC))
	updateCloudRunMinInstances(filteredServices)
}

func updateCloudRunMinInstances(services []Service) {
	url := "https://run.googleapis.com/v2/projects/" + IntProjectId + "/locations/" + os.Getenv(REGION) + "/services/" + ServiceName + "?update_mask=scaling.minInstanceCount"
	var wg sync.WaitGroup
	for _, service := range services {
		// wait group count is equivalent to number of services/thread
		wg.Add(1)
		// using Go routine to concurrently trigger updateCloudRunMinInstance
		go updateCloudRunMinInstance(url, service.name, service.minInstances, &wg)
	}
	// wait groups are waiting till all services are done
	wg.Wait()
	// Here all services are done updating
	fmt.Printf("Total %d services executed \n", len(services))
}

func updateCloudRunMinInstance(url string, serviceName string, minInstances int, wg *sync.WaitGroup) {
	fmt.Println(serviceName + " : " + strconv.Itoa(minInstances))

	// Once this function is completed, wg.Done() will be trigger, indicating thread is done
	defer wg.Done()
	url = strings.Replace(url, ServiceName, serviceName, 1)
	payload := []byte(fmt.Sprintf(`{ "scaling": { "minInstanceCount": %d }}`, minInstances))
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Printf("Service: %s Error creating request:\n %s", serviceName, err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv(Token))

	// Create a new HTTP client and execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request for Service %s :\n %s", serviceName, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Request failed for Service %s with status code: %d \n", serviceName, resp.StatusCode)
		return
	}

	fmt.Printf("Service %s successfully updated.\n", serviceName)
}

func validateEnv() {
	_, ok := os.LookupEnv(Token)
	if !ok {
		log.Fatalf("Token not set\n")
	}
	_, ok = os.LookupEnv(REGION)
	if !ok {
		log.Fatalf("Region not set\n")
	}
	traffic := os.Getenv(TRAFFIC)
	if len(traffic) == 0 {
		log.Fatalf("Traffic not set\n")
	}
	if traffic != LowTraffic && traffic != HighTraffic {
		log.Fatalf("Traffic value is invalid \n")
	}
}

func getWhiteListedServices(traffic string) []Service {
	file, err := os.Open(servicefile)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer file.Close()
	minInstances := 1 // Set all service instance to 1 for high traffic hours
	if traffic == LowTraffic {
		minInstances = 0 // Set all service instance to 0 for low traffic hours
	}
	var services []Service

	// Create a scanner to read the file line by line
	reader := csv.NewReader(file)
	// discarding first header row
	reader.Read()
	for {
		// Read a single line from the file
		record, err := reader.Read()
		// Check for end of file
		if err != nil {
			//If it's the end of the file, break the loop
			if err.Error() == "EOF" {
				break
			}
		}
		if len(record) == 1 {
			services = append(services, Service{record[0], minInstances})
		} else if len(record) == 3 {
			if traffic == LowTraffic {
				services = append(services, Service{record[0], parseString(record[1])})
			} else {
				services = append(services, Service{record[0], parseString(record[2])})
			}
		} else {
			log.Fatalf("Error parsing line: %d \n", len(record))
		}
	}
	return services
}

func parseString(val string) int {
	intVal, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		fmt.Errorf("Error reading min instance for '%s' \n", val)
	}
	return intVal
}
