package main

import (
	"fmt"
	"os"
	"strconv"
)

func amIRunningOnLambda() bool {
	lfname := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	lfmem := os.Getenv("AWS_LAMBDA_FUNCTION_MEMORY_SIZE")

	region := os.Getenv("AWS_REGION")

	if lfname != "" && lfmem != "" {
		fmt.Printf("Seems like running on Lambda: Funcname %s, Mem %s, Region %s\n", lfname, lfmem, region)
		return true
	}
	return false
}

func getEnvS(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvF(key string, fallback float64) float64 {
	if value, ok := os.LookupEnv(key); ok {
		f, _ := strconv.ParseFloat(value, 64) //not production ready, no error handling
		return f
	}
	return fallback
}

func getEnvI(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		i, _ := strconv.Atoi(value) //not production ready, no error handling
		return i
	}
	return fallback
}
