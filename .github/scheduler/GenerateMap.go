package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

const TRAFFIC = "TRAFFIC"
const LowTraffic = "low"
const HighTraffic = "high"
const IntProjectId = "int"


func main() {
	//os.Setenv(TRAFFIC, "high") // Uncomment for local debugging
	fmt.Println("Updating Min Instances...")
	validateEnv()
	data:= getWhiteListedServicesMap(os.Getenv(TRAFFIC))
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling map to JSON:", err)
		os.Exit(1)  //# Exit with non-zero code on error
	}
	fmt.Println(string(jsonData))
}



func getWhiteListedServicesMap(traffic string) map[string]int {
	file, err := os.Open("./.github/scheduler/SchedulerServices.csv")
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer file.Close()
	minInstances := 1 // Set all service instance to 1 for high traffic hours
	if traffic == LowTraffic {
		minInstances = 0 // Set all service instance to 0 for low traffic hours
	}
	serviceMap := make(map[string]int)

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
			serviceMap[record[0]] = minInstances
			//services = append(services, Service{record[0], minInstances})
		} else if len(record) == 3 {
			if traffic == LowTraffic {
				serviceMap[record[0]] = parseString(record[1])
			} else {
				serviceMap[record[0]] = parseString(record[2])
			}
		} else {
			log.Fatalf("Error parsing line: %d \n", len(record))
		}
	}
	return serviceMap
}

func parseString(val string) int {
	intVal, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		fmt.Errorf("Error reading min instance for '%s' \n", val)
	}
	return intVal
}

func validateEnv() {
	traffic := os.Getenv(TRAFFIC)
	if len(traffic) == 0 {
		log.Fatalf("Traffic not set\n")
	}
	if traffic != LowTraffic && traffic != HighTraffic {
		log.Fatalf("Traffic value is invalid \n")
	}
}
