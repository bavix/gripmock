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

type stub struct {
	Service string `json:"service"`
	Method  string `json:"method"`
	Input   struct {
		Equals   map[string]string `json:"equals,omitempty"`
		Contains map[string]string `json:"contains,omitempty"`
		Matches  map[string]string `json:"matches,omitempty"`
	} `json:"input"`
	Output struct {
		Data map[string]string `json:"data"`
	} `json:"output"`
}

func generateDataset(count int, base, csvPath string) error {
	stubs := make([]stub, count)
	contains := make([]stub, count)
	patterns := make([]stub, count)
	order := make([]int, count)

	for i := range count {
		name := fmt.Sprintf("user-%06d", i)

		data := map[string]string{"message": fmt.Sprintf("Hello, %s!", name)}

		e := stub{Service: "Greeter", Method: "SayHello"}
		e.Input.Equals = map[string]string{"name": name}
		e.Output.Data = data
		stubs[i] = e

		c := stub{Service: "Greeter", Method: "SayHello"}
		c.Input.Contains = map[string]string{"name": name}
		c.Output.Data = data
		contains[i] = c

		m := stub{Service: "Greeter", Method: "SayHello"}
		m.Input.Matches = map[string]string{"name": regexFor(name)}
		m.Output.Data = data
		patterns[i] = m

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

	for path, set := range map[string][]stub{
		base + "-equals.json":   stubs,
		base + "-contains.json": contains,
		base + "-matches.json":  patterns,
	} {
		if err := writeStubs(path, set); err != nil {
			return err
		}
	}

	return nil
}

func regexFor(name string) string {
	last := len(name) - 1

	return "^" + name[:last] + "[" + name[last:] + "]$"
}

func writeStubs(path string, set []stub) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	if err := json.NewEncoder(f).Encode(set); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}

	return nil
}
