package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"pasteguard/detector"
	"pasteguard/server"
)

type Result struct {
	OverallRisk   string             `json:"overall_risk"`
	RiskRationale string             `json:"risk_rationale"`
	Findings      []detector.Finding `json:"findings"`
}

func main() {
	// Check if "serve" command is provided
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		runServer()
		return
	}

	// Otherwise, run CLI mode
	runCLI()
}

func runServer() {
	// Parse serve command flags
	serveFlags := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := serveFlags.String("addr", ":8787", "Address to listen on")
	serveFlags.Parse(os.Args[2:])

	// Create and start server
	srv := server.NewServer()
	if err := srv.Start(*addr); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
}

func runCLI() {
	// Manually parse --text flag to handle empty strings
	// Go's flag package doesn't handle --text "" well
	var textValue string
	textFlagProvided := false
	
	// Check if --text is in args
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "--text" || arg == "-text" {
			textFlagProvided = true
			// Check if next argument exists
			if i+1 < len(args) {
				textValue = args[i+1]
				// Remove --text and its value from args
				newArgs := make([]string, 0, len(args)-2)
				newArgs = append(newArgs, args[:i]...)
				if i+2 < len(args) {
					newArgs = append(newArgs, args[i+2:]...)
				}
				os.Args = append([]string{os.Args[0]}, newArgs...)
			} else {
				// No value provided, treat as empty
				textValue = ""
				// Remove --text from args
				newArgs := make([]string, 0, len(args)-1)
				newArgs = append(newArgs, args[:i]...)
				if i+1 < len(args) {
					newArgs = append(newArgs, args[i+1:]...)
				}
				os.Args = append([]string{os.Args[0]}, newArgs...)
			}
			break
		} else if len(arg) >= 7 && arg[:7] == "--text=" {
			textFlagProvided = true
			textValue = arg[7:]
			// Remove this arg from os.Args
			newArgs := make([]string, 0, len(args)-1)
			newArgs = append(newArgs, args[:i]...)
			if i+1 < len(args) {
				newArgs = append(newArgs, args[i+1:]...)
			}
			os.Args = append([]string{os.Args[0]}, newArgs...)
			break
		} else if len(arg) >= 6 && arg[:6] == "-text=" {
			textFlagProvided = true
			textValue = arg[6:]
			// Remove this arg from os.Args
			newArgs := make([]string, 0, len(args)-1)
			newArgs = append(newArgs, args[:i]...)
			if i+1 < len(args) {
				newArgs = append(newArgs, args[i+1:]...)
			}
			os.Args = append([]string{os.Args[0]}, newArgs...)
			break
		}
	}

	// Parse remaining flags (if any)
	flag.Parse()

	var input string
	var err error

	if textFlagProvided {
		// --text flag was provided, use it (even if empty string)
		input = textValue
	} else {
		// Read from stdin
		data, readErr := io.ReadAll(os.Stdin)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", readErr)
			os.Exit(1)
		}
		input = string(data)
	}

	// Create engine and analyze
	engine := detector.NewEngine()
	result := engine.Analyze(input)

	// Build output
	output := Result{
		OverallRisk:   result.OverallRisk,
		RiskRationale: result.RiskRationale,
		Findings:      result.Findings,
	}

	// Output JSON
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
