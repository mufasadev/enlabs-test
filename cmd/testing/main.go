package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

var URL, _ = os.LookupEnv("API_URL")
var PORT, _ = os.LookupEnv("API_PORT")
var apiURL = fmt.Sprintf("http://%s:%s/api/v1/users/f60ae2e1-ee72-4a6a-bef2-7cde5c83782f", URL, PORT)
var transactionsURL = apiURL + "/transactions"
var balanceURL = apiURL + "/balance"

const (
	workers  = 10
	duration = 30 * time.Second
)

var sourceTypes = []string{"game", "server", "payment"}

type Transaction struct {
	State         string `json:"state"`
	Amount        string `json:"amount"`
	TransactionID string `json:"transactionId"`
}

func main() {
	var wg sync.WaitGroup
	wg.Add(workers + 1)
	var transactionResponse interface{}
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			start := time.Now()
			for time.Since(start) < duration {
				resp, err := sendTransaction()
				if err != nil && resp != nil {
					fmt.Println("Error sending transaction:", err)
				}

				if resp != nil {
					err = json.NewDecoder(resp.Body).Decode(&transactionResponse)
					if err != nil {
						resp.Body.Close()
						fmt.Printf("error decoding transaction response: %v", err)
					}

					fmt.Printf("Transaction sent. Status code: %d, Message: %v\n", resp.StatusCode, transactionResponse)
					resp.Body.Close()
				}

				time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
			}
		}()
	}

	go func() {
		defer wg.Done()
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			printBalance()
		}
	}()

	wg.Wait()
	printBalance()
}

func sendTransaction() (*http.Response, error) {
	transaction := createTransaction()
	data, err := json.Marshal(transaction)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, transactionsURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Source-Type", sourceTypes[rand.Intn(len(sourceTypes))])

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("wrong status code: %d", resp.StatusCode)
	}
	return resp, nil
}

func createTransaction() Transaction {
	state := "lost"
	if rand.Float64() < 0.5 {
		state = "win"
	}

	amount := rand.Float64()*1000 + 1
	amountStr := fmt.Sprintf("%.2f", amount)

	transactionID := uuid.New().String()
	if rand.Float64() < 0.05 {
		transactionID = uuid.New().String()[:10]
	}

	return Transaction{
		State:         state,
		Amount:        amountStr,
		TransactionID: transactionID,
	}
}

func printBalance() {
	resp, err := http.Get(balanceURL)
	if err != nil {
		fmt.Println("Error getting balance:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Wrong status code:", resp.StatusCode)
		return
	}

	var balanceResponse struct {
		Balance float64 `json:"balance"`
	}
	err = json.NewDecoder(resp.Body).Decode(&balanceResponse)
	if err != nil {
		fmt.Println("Error decoding balance:", err)
		return
	}

	fmt.Printf("User balance: %.2f\n", balanceResponse.Balance)
}
