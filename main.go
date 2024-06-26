package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Receipt struct {
	Retailer     string  `json:"retailer"`
	PurchaseDate string  `json:"purchaseDate"`
	PurchaseTime string  `json:"purchaseTime"`
	Total        string  `json:"total"`
	Items        []Item  `json:"items"`
}

type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

var receipts = make(map[string]Receipt)
var points = make(map[string]int)

func main() {
	http.HandleFunc("/receipts/process", processReceipts)
	http.HandleFunc("/receipts/", getPoints)
	http.HandleFunc("/receipts", listReceipts)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func processReceipts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var receipt Receipt
	err := json.NewDecoder(r.Body).Decode(&receipt)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	receipts[id] = receipt
	points[id] = calculatePoints(receipt)

	response := map[string]string{"id": id}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getPoints(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 3 || pathParts[2] != "points" {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	id := pathParts[1]
	point, ok := points[id]
	if !ok {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	response := map[string]int{"points": point}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func calculatePoints(receipt Receipt) int {
	points := 0

	// 1 point for every alphanumeric character in the retailer name
	alnumRegex := regexp.MustCompile(`[a-zA-Z0-9]`)
	retailerPoints := len(alnumRegex.FindAllString(receipt.Retailer, -1))
	points += retailerPoints

	// 50 points if the total is a round dollar amount with no cents
	total, _ := strconv.ParseFloat(receipt.Total, 64)
	if total == float64(int(total)) {
		points += 50
	}

	// 25 points if the total is a multiple of 0.25
	if int(total*100)%25 == 0 {
		points += 25
	}

	// 5 points for every two items on the receipt
	itemPairPoints := (len(receipt.Items) / 2) * 5
	points += itemPairPoints

	// Points based on item description length and price
	for _, item := range receipt.Items {
		trimmedDescription := strings.TrimSpace(item.ShortDescription)
		if len(trimmedDescription)%3 == 0 {
			price, _ := strconv.ParseFloat(item.Price, 64)
			itemPoints := int(math.Ceil(price * 0.2))
			points += itemPoints
		}
	}

	// 6 points if the purchase date day is odd
	date, _ := time.Parse("2006-01-02", receipt.PurchaseDate)
	if date.Day()%2 != 0 {
		points += 6
	}

	// 10 points if the purchase time is between 2:00 PM and 4:00 PM
	time, _ := time.Parse("15:04", receipt.PurchaseTime)
	if time.Hour() >= 14 && time.Hour() < 16 {
		points += 10
	}

	return points
}

// New handler to list all receipts
func listReceipts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(receipts)
}