package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"os"

	"repro/proto/github.com/0xPolygonHermez/zkevm-node/state/runtime/executor"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func main() {
	dumpDir := filepath.Join("verifier_dumps")
	payloadFiles, err := os.ReadDir(dumpDir)
	if err != nil {
		fmt.Printf("Failed to read dump directory: %v\n", err)
		return
	}

	sort.Slice(payloadFiles, func(i, j int) bool {
		batchNumI := extractBatchNumber(payloadFiles[i].Name())
		batchNumJ := extractBatchNumber(payloadFiles[j].Name())
		return batchNumI < batchNumJ
	})

	executorUrls := []string{
		"34.175.105.40:50071",
		"34.175.232.61:50071",
		"34.175.237.160:50071",
		"34.175.42.120:50071",
		"34.175.236.212:50071",
		"34.175.167.26:50071",
		"51.210.116.237:50071",
		//"34.175.73.226:50071", - currently down
	}

	executors := make([]*Executor, len(executorUrls))
	for i, url := range executorUrls {
		executors[i], err = NewExecutor(url, 5000*time.Millisecond)
		if err != nil {
			fmt.Printf("Failed to create executor: %v\n", err)
			return
		}
	}

	for _, fileInfo := range payloadFiles {
		if !strings.Contains(fileInfo.Name(), "-payload.json") {
			continue
		}

		payloadPath := filepath.Join(dumpDir, fileInfo.Name())
		payload, request, err := loadPayloadandRequest(payloadPath)
		if err != nil {
			fmt.Printf("Error loading payload and request from %s: %v\n", payloadPath, err)
			continue
		}

		for _, e := range executors {
			_, resp, err := e.VerifyTest(payload, request)
			if err != nil {
				fmt.Printf("%s: Error verifying payload %s: %v\n", e.grpcUrl, payloadPath, err)
			}

			responseJson, _ := json.MarshalIndent(resp, "", "  ")
			outFile := fmt.Sprintf("%s/output_%s_%s.json", "output", extractBatchNumberString(fileInfo.Name()), e.grpcUrl)
			if err := os.WriteFile(outFile, []byte(responseJson), 0644); err != nil {
				fmt.Printf("Failed to write response to file: %v\n", err)
			}
		}
	}
}

func extractBatchNumber(filename string) int {
	parts := strings.Split(filename, "-")
	for _, part := range parts {
		if strings.HasPrefix(part, "batch_") {
			batchStr := strings.TrimPrefix(part, "batch_")
			batchNum, err := strconv.Atoi(batchStr)
			if err == nil {
				return batchNum
			}
			break
		}
	}
	return 0
}

func extractBatchNumberString(filename string) string {
	parts := strings.Split(filename, "-")
	for _, part := range parts {
		if strings.HasPrefix(part, "batch_") {
			return strings.TrimPrefix(part, "batch_")
		}
	}
	return "unknown"
}

func responsesAreEqual(t *testing.T, resp1, resp2 *executor.ProcessBatchResponseV2) bool {
	resp1Copy := *resp1
	resp2Copy := *resp2

	resp1Copy.FlushId, resp2Copy.FlushId = 0, 0
	resp1Copy.ProverId, resp2Copy.ProverId = "", ""
	resp1Copy.CntPoseidonHashes, resp2Copy.CntPoseidonHashes = 0, 0
	resp1Copy.StoredFlushId, resp2Copy.StoredFlushId = 0, 0

	o := cmpopts.IgnoreUnexported(
		executor.ProcessBatchResponseV2{},
		executor.ProcessBlockResponseV2{},
		executor.ProcessTransactionResponseV2{},
		executor.InfoReadWriteV2{},
	)
	diff := cmp.Diff(resp1Copy, resp2Copy, o)
	if diff != "" {
		t.Errorf("Objects differ: %v", diff)
		return false
	}

	return true
}

func loadPayloadandRequest(filePath string) (*Payload, *VerifierRequest, error) {
	payloadData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}
	var payload Payload
	if err := json.Unmarshal(payloadData, &payload); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	requestData, err := os.ReadFile(strings.Replace(filePath, "-payload.json", "-request.json", 1))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	var request VerifierRequest
	if err := json.Unmarshal(requestData, &request); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	return &payload, &request, nil
}
