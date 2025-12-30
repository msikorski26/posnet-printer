package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Transaction reprezentuje transakcję z CSV
type Transaction struct {
	Date   string // YYYY-MM-DD
	Amount int    // kwota w groszach
}

// ParseCSVFile parsuje jeden plik CSV
func ParseCSVFile(path string) ([]Transaction, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("błąd otwierania pliku %s: %w", path, err)
	}
	defer file.Close()

	var transactions []Transaction
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // pusta linia
		}

		// Format: 2025-12-01; 197,99
		parts := strings.Split(line, ";")
		if len(parts) != 2 {
			// Może być błąd w linii, ale kontynuujemy
			continue
		}

		date := strings.TrimSpace(parts[0])
		amountStr := strings.TrimSpace(parts[1])

		// Konwertuj kwotę: "197,99" -> 19799 groszy
		amountStr = strings.ReplaceAll(amountStr, ",", ".")
		amountFloat, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			fmt.Printf("Ostrzeżenie: nie można sparsować kwoty w linii %d: %s\n", lineNum, line)
			continue
		}

		amountGr := int(amountFloat*100 + 0.5) // zaokrąglenie

		transactions = append(transactions, Transaction{
			Date:   date,
			Amount: amountGr,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("błąd czytania pliku %s: %w", path, err)
	}

	return transactions, nil
}

// ParseCSVDirectory parsuje wszystkie pliki CSV w katalogu
func ParseCSVDirectory(dirPath string) ([]Transaction, error) {
	files, err := filepath.Glob(filepath.Join(dirPath, "*.csv"))
	if err != nil {
		return nil, fmt.Errorf("błąd wyszukiwania plików CSV: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("nie znaleziono plików CSV w katalogu %s", dirPath)
	}

	var allTransactions []Transaction
	for _, file := range files {
		transactions, err := ParseCSVFile(file)
		if err != nil {
			fmt.Printf("Ostrzeżenie: błąd parsowania %s: %v\n", file, err)
			continue
		}
		allTransactions = append(allTransactions, transactions...)
	}

	return allTransactions, nil
}

// GroupByDate grupuje transakcje po datach
func GroupByDate(transactions []Transaction) map[string][]Transaction {
	grouped := make(map[string][]Transaction)
	for _, t := range transactions {
		grouped[t.Date] = append(grouped[t.Date], t)
	}
	return grouped
}

// GetUniqueDates zwraca posortowaną listę unikalnych dat
func GetUniqueDates(transactions []Transaction) []string {
	dateSet := make(map[string]bool)
	for _, t := range transactions {
		dateSet[t.Date] = true
	}

	var dates []string
	for date := range dateSet {
		dates = append(dates, date)
	}

	// Sortowanie dat (proste sortowanie stringów działa dla YYYY-MM-DD)
	for i := 0; i < len(dates); i++ {
		for j := i + 1; j < len(dates); j++ {
			if dates[i] > dates[j] {
				dates[i], dates[j] = dates[j], dates[i]
			}
		}
	}

	return dates
}
