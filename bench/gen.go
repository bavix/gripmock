package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
)

const seed = 0x9E3779B97F4A7C15

func shuffle(order []int) {
	state := uint64(seed)

	next := func() uint64 {
		state += 0x9E3779B97F4A7C15
		z := state
		z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
		z = (z ^ (z >> 27)) * 0x94D049BB133111EB

		return z ^ (z >> 31)
	}

	for i := len(order) - 1; i > 0; i-- {
		j := int(next() % uint64(i+1))
		order[i], order[j] = order[j], order[i]
	}
}

func generateDataset(count int, stubsPath, csvPath string) error {
	type stub struct {
		Service string `json:"service"`
		Method  string `json:"method"`
		Input   struct {
			Equals map[string]string `json:"equals"`
		} `json:"input"`
		Output struct {
			Data map[string]string `json:"data"`
		} `json:"output"`
	}

	stubs := make([]stub, count)
	order := make([]int, count)

	for i := range count {
		name := fmt.Sprintf("user-%06d", i)

		s := stub{Service: "Greeter", Method: "SayHello"}
		s.Input.Equals = map[string]string{"name": name}
		s.Output.Data = map[string]string{"message": fmt.Sprintf("Hello, %s!", name)}
		stubs[i] = s
		order[i] = i
	}

	shuffle(order)

	csvFile, err := os.Create(csvPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", csvPath, err)
	}
	defer func() { _ = csvFile.Close() }()

	csvWriter := csv.NewWriter(csvFile)
	if err := csvWriter.Write([]string{"name"}); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	for _, idx := range order {
		if err := csvWriter.Write([]string{fmt.Sprintf("user-%06d", idx)}); err != nil {
			return fmt.Errorf("write csv row %d: %w", idx, err)
		}
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}

	stubsFile, err := os.Create(stubsPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", stubsPath, err)
	}
	defer func() { _ = stubsFile.Close() }()

	enc := json.NewEncoder(stubsFile)
	if err := enc.Encode(stubs); err != nil {
		return fmt.Errorf("encode stubs: %w", err)
	}

	return nil
}
