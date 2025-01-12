package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Structs to handle API response for test cases
type APIResponse struct {
	Status bool `json:"status"`
	Result struct {
		Total    int      `json:"total"`
		Entities []Entity `json:"entities"`
	} `json:"result"`
}

type Entity struct {
	ID            int           `json:"id"`
	Title         string        `json:"title"`
	CustomFields  []CustomField `json:"custom_fields"`
	CustomFieldID int           // Extracted custom field value
}

type CustomField struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
}

// Structs for test results
type TestRun struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Status int    `json:"status"`
}

type Step struct {
	Status      interface{}  `json:"status"`
	Comment     string       `json:"comment"`
	Attachments []Attachment `json:"attachments"`
	Position    int          `json:"position"`
}

type Attachment struct {
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

type Result struct {
	Hash        string       `json:"hash"`
	Comment     string       `json:"comment"`
	StackTrace  string       `json:"stacktrace"`
	Steps       []Step       `json:"steps"`
	Status      string       `json:"status"`
	CaseID      int          `json:"case_id"`
	Attachments []Attachment `json:"attachments"`
	Duration    int          `json:"time_spent_ms"`
}

type BulkResultRequest struct {
	Results []BulkResult `json:"results"`
}

type BulkResult struct {
	CaseID     string `json:"case_id"`
	Stacktrace string `json:"stacktrace"`
	Comment    string `json:"comment"`
	Steps      []Step `json:"steps"`
	Status     string `json:"status"`
	Duration   int    `json:"time_ms"`
}

// Function to fetch all test cases
func fetchTestCases(apiURL, token string) ([]Entity, error) {
	var allCases []Entity
	offset := 0
	limit := 100

	for {
		url := fmt.Sprintf("%s&offset=%d", apiURL, offset)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Add("accept", "application/json")
		req.Header.Add("Token", token)

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		body, _ := io.ReadAll(res.Body)
		var response APIResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, err
		}

		allCases = append(allCases, response.Result.Entities...)
		if len(response.Result.Entities) < limit {
			break
		}
		offset += limit
	}
	return allCases, nil
}

func fetchTestResults(project string, token string, runID int) ([]Result, error) {
	var allResults []Result
	offset := 0
	limit := 100

	for {
		url := fmt.Sprintf("https://api.qase.io/v1/result/%s?run=%d&limit=%d&offset=%d", project, runID, limit, offset)
		fmt.Println("Fetching results from URL:", url)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Add("accept", "application/json")
		req.Header.Add("Token", token)

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		fmt.Println("API Response Body:", string(body))

		var result struct {
			Result struct {
				Entities []Result `json:"entities"`
			} `json:"result"`
		}

		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		allResults = append(allResults, result.Result.Entities...)
		if len(result.Result.Entities) < limit {
			break
		}
		offset += limit
	}
	return allResults, nil
}

// Load the CSV mapping from in-memory data
func loadCSVMapping(csvData string) (map[int]int, error) {
	// Read the CSV data from a string (in-memory)
	reader := csv.NewReader(strings.NewReader(csvData))
	reader.FieldsPerRecord = -1

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	// Initialize the mapping
	mapping := make(map[int]int)

	// Skip the header row
	if len(records) > 0 {
		records = records[1:]
	}

	// Loop through the records (starting from the second row)
	for _, record := range records {
		// Convert Source and Target IDs to integers
		sourceCaseID, err := strconv.Atoi(record[0])
		if err != nil {
			return nil, fmt.Errorf("invalid source case ID: %v", err)
		}
		targetCaseID, err := strconv.Atoi(record[2])
		if err != nil {
			return nil, fmt.Errorf("invalid target case ID: %v", err)
		}

		// Add to the mapping
		mapping[sourceCaseID] = targetCaseID
	}

	fmt.Printf("Loaded CSV Mapping: %v\n", mapping)

	return mapping, nil
}

// Prepare bulk results from the fetched test results and CSV mapping
func prepareBulkResults(results []Result, mapping map[int]int) ([]BulkResult, error) {
	var bulkResults []BulkResult

	// Loop through the results to prepare bulk results
	for _, result := range results {
		fmt.Printf("Processing Result: %v\n", result)

		// Check if the result has a corresponding entry in the CSV mapping
		targetCaseID, exists := mapping[result.CaseID]
		if !exists {
			// If there's no mapping for the case, skip it
			fmt.Printf("No mapping found for Source CaseID: %d\n", result.CaseID)
			continue
		}

		// Prepare the result for bulk insertion
		bulkResult := BulkResult{
			CaseID:     fmt.Sprintf("%d", targetCaseID),
			Stacktrace: result.StackTrace,
			Comment:    result.Comment,
			Duration:   result.Duration,
			Status:     mapStatus(result.Status),
			Steps:      mapSteps(result.Steps),
		}

		// Add attachments in markdown format if they exist
		// for _, attachment := range result.Attachments {
		// 	bulkResult.Comment += fmt.Sprintf("\n\n[%s](%s)", attachment.Filename, attachment.URL)
		// }

		for _, attachment := range result.Attachments {
			// Decode the filename to handle special characters
			decodedFilename, err := url.QueryUnescape(attachment.Filename)
			if err != nil {
				// If there's an error decoding the filename, log it and use the original filename
				fmt.Printf("Error decoding filename %s: %v\n", attachment.Filename, err)
				decodedFilename = attachment.Filename
			}

			// Add the attachment in markdown format
			bulkResult.Comment += fmt.Sprintf("\n\n[%s](%s)", decodedFilename, attachment.URL)
		}

		bulkResults = append(bulkResults, bulkResult)
	}

	// Ensure we have results to send
	if len(bulkResults) == 0 {
		return nil, fmt.Errorf("no valid results to send")
	}

	return bulkResults, nil
}

// Map the status codes to the required status for the bulk insert
func mapStatus(status string) string {
	// Map the status string to the correct status text for the bulk request
	switch status {
	case "1":
		return "passed"
	case "2":
		return "failed"
	case "3":
		return "blocked"
	case "5":
		return "skipped"
	case "passed":
		return "passed"
	case "failed":
		return "failed"
	case "invalid":
		return "invalid"
	case "blocked":
		return "blocked"
	case "skipped":
		return "skipped"
	default:
		return "unknown"
	}
}

// Map steps from the fetched result and convert step status to string values
func mapSteps(steps []Step) []Step {
	var mappedSteps []Step
	for _, step := range steps {
		// Use type assertion to check if step.Status is string or int
		var statusStr string
		switch v := step.Status.(type) {
		case string:
			statusStr = v
		case float64:
			statusStr = fmt.Sprintf("%d", int(v))
		default:
			statusStr = "unknown"
		}

		// Map the status string using the mapStepStatus function
		mappedStep := Step{
			Status:  mapStepStatus(statusStr),
			Comment: step.Comment,
		}

		mappedSteps = append(mappedSteps, mappedStep)
	}
	return mappedSteps
}

// Convert step status string to corresponding string value
func mapStepStatus(status string) string {
	// Convert string to integer
	statusInt, err := strconv.Atoi(status)
	if err != nil {
		return "unknown"
	}

	switch statusInt {
	case 0:
		return "skipped"
	case 1:
		return "passed"
	case 2:
		return "failed"
	case 3:
		return "invalid"
	case 5:
		return "skipped"
	default:
		return "unknown"
	}
}

// Create the bulk results in QASE, with multiple requests if there are more than 1500 results
func bulkCreateResults(project string, runID int, results []BulkResult, token string) error {
	// Split results into chunks of 200 or fewer
	chunkSize := 200
	for i := 0; i < len(results); i += chunkSize {
		end := i + chunkSize
		if end > len(results) {
			end = len(results)
		}

		// Create payload for this chunk
		payloadData := fmt.Sprintf("{\"results\":%s}", toJSON(results[i:end]))

		// Log the exact payload being sent to the API for debugging purposes
		fmt.Println("Payload being sent to the API:")
		fmt.Println(payloadData)

		// Prepare the request
		url := fmt.Sprintf("https://api.qase.io/v1/result/%s/%d/bulk", project, runID)
		payload := strings.NewReader(payloadData)

		req, _ := http.NewRequest("POST", url, payload)
		req.Header.Add("accept", "application/json")
		req.Header.Add("content-type", "application/json")
		req.Header.Add("Token", token)

		// Make the request to the API
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		// Read the response body
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}

		fmt.Println("API Response Body:", string(body))
	}

	return nil
}

// Helper function to convert any value to a JSON string
func toJSON(value interface{}) string {
	data, _ := json.Marshal(value)
	return string(data)
}

// Function to asynchronously write mapping to CSV file
func writeMappingToCSV(mapping map[int]int, filename string, wg *sync.WaitGroup) {
	defer wg.Done()
	// Create or overwrite the file
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating mapping file:", err)
		return
	}
	defer file.Close()

	// Write the CSV data
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	err = writer.Write([]string{"Source Case ID", "Target Case ID"})
	if err != nil {
		fmt.Println("Error writing header to mapping file:", err)
		return
	}

	// Write the mappings
	for sourceID, targetID := range mapping {
		err := writer.Write([]string{strconv.Itoa(sourceID), strconv.Itoa(targetID)})
		if err != nil {
			fmt.Println("Error writing mapping to file:", err)
			return
		}
	}

	fmt.Println("Mapping has been written to", filename)
}

func main() {
	// Retrieve environment variables or fallbacks
	apiToken, sourceProject, targetProject, customFieldIDStr, sourceRunIDStr, targetRunIDStr, err := getVariables()
	if err != nil {
		fmt.Println("Error retrieving variables:", err)
		os.Exit(1)
	}

	// Convert custom field ID to int
	customFieldID, err := strconv.Atoi(customFieldIDStr)
	if err != nil {
		fmt.Println("Error parsing QASE_CF_ID:", err)
		return
	}

	// Convert run IDs to int
	sourceRunID, err := strconv.Atoi(sourceRunIDStr)
	if err != nil {
		fmt.Println("Error converting QASE_SOURCE_RUN:", err)
		return
	}

	targetRunID, err := strconv.Atoi(targetRunIDStr)
	if err != nil {
		fmt.Println("Error converting QASE_TARGET_RUN:", err)
		return
	}

	// Define API URLs
	sourceURL := fmt.Sprintf("https://api.qase.io/v1/case/%s?limit=100", sourceProject)
	targetURL := fmt.Sprintf("https://api.qase.io/v1/case/%s?limit=100", targetProject)

	// Fetch test cases from source and target projects
	sourceCases, err := fetchTestCases(sourceURL, apiToken)
	if err != nil {
		fmt.Println("Error fetching source cases:", err)
		return
	}
	targetCases, err := fetchTestCases(targetURL, apiToken)
	if err != nil {
		fmt.Println("Error fetching target cases:", err)
		return
	}

	// Map target cases by custom field value
	targetMap := make(map[int]Entity)
	for _, tCase := range targetCases {
		for _, field := range tCase.CustomFields {
			if field.ID == customFieldID {
				fieldValue, _ := strconv.Atoi(field.Value)
				targetMap[fieldValue] = tCase
			}
		}
	}

	// Prepare CSV data in memory
	var buffer bytes.Buffer
	csvWriter := csv.NewWriter(&buffer)

	// Write header and data to the buffer
	csvData := [][]string{{"Source Case ID", "Source Title", "Target Case ID", "Target Title"}}
	for _, sCase := range sourceCases {
		if matchingTarget, found := targetMap[sCase.ID]; found {
			row := []string{
				strconv.Itoa(sCase.ID),
				sCase.Title,
				strconv.Itoa(matchingTarget.ID),
				matchingTarget.Title,
			}
			csvData = append(csvData, row)
		}
	}

	for _, row := range csvData {
		if err := csvWriter.Write(row); err != nil {
			fmt.Println("Error writing to in-memory CSV:", err)
			return
		}
	}
	csvWriter.Flush()

	// Load CSV mapping from in-memory CSV content
	csvDataString := buffer.String()
	mapping, err := loadCSVMapping(csvDataString)
	if err != nil {
		fmt.Println("Error loading CSV mapping:", err)
		return
	}

	// Fetch test results from source run ID
	results, err := fetchTestResults(sourceProject, apiToken, sourceRunID)
	if err != nil {
		fmt.Println("Error fetching test results:", err)
		return
	}

	// Prepare results for bulk creation
	bulkResults, err := prepareBulkResults(results, mapping)
	if err != nil {
		fmt.Println("Error preparing bulk results:", err)
		return
	}

	// Log the target run ID being used for bulk creation
	fmt.Printf("Using target run ID: %d for bulk creation\n", targetRunID)

	// Asynchronously write the mapping to CSV
	var wg sync.WaitGroup
	wg.Add(1)
	go writeMappingToCSV(mapping, "mapping.csv", &wg)

	// Send bulk results to the target run
	err = bulkCreateResults(targetProject, targetRunID, bulkResults, apiToken)
	if err != nil {
		fmt.Println("Error creating bulk results:", err)
	}

	// Wait for the asynchronous write operation to finish
	wg.Wait()
}
