package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

func main() {
	var (
		configPath    = flag.String("config", "config.json", "Ścieżka do pliku konfiguracji")
		dataPath      = flag.String("data", "data.json", "Ścieżka do pliku danych (produkty)")
		csvPath       = flag.String("csv", "", "Ścieżka do pliku CSV (np. reports/01.csv) lub katalogu z plikami CSV")
		createCfg     = flag.Bool("create-config", false, "Utwórz przykładowy plik konfiguracji i zakończ")
		dryRun        = flag.Bool("dry-run", false, "Tryb testowy - nie łącz się z drukarką, tylko wyświetl co zostałoby wydrukowane")
		dailyReport          = flag.String("daily-report", "", "Wydrukuj raport dobowy (zawsze dla bieżącego dnia)")
		monthlyReport        = flag.String("monthly-report", "", "Wydrukuj raport miesięczny dla podanej daty (format: YYYY-MM-DD, brana pod uwagę tylko miesiąc i rok) lub puste dla bieżącego miesiąca")
		monthlyReportSummary = flag.Bool("monthly-report-summary", false, "Raport miesięczny w wersji skróconej (podsumowanie)")
	)
	flag.Parse()

	if *createCfg {
		cfg := CreateExampleConfig()
		if err := cfg.SaveConfig(*configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Błąd zapisu przykładowej konfiguracji: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Utworzono przykładową konfigurację: %s\n", *configPath)

		data := CreateExampleData()
		if err := data.SaveData(*dataPath); err != nil {
			fmt.Fprintf(os.Stderr, "Błąd zapisu przykładowych danych: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Utworzono przykładowe dane produktów: %s\n", *dataPath)
		fmt.Println("Edytuj pliki i dostosuj ustawienia przed użyciem.")
		return
	}

	if *dailyReport != "" || *monthlyReport != "" {
		fmt.Printf("→ Wczytuję konfigurację z %s...\n", *configPath)
		cfg, err := LoadConfig(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Błąd wczytywania konfiguracji: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Konfiguracja wczytana")

		if *dryRun {
			fmt.Println("⚠ TRYB TESTOWY - symulacja bez drukarki")
			if *dailyReport != "" {
				fmt.Println("✓ [SYMULACJA] Raport dobowy")
			}
			if *monthlyReport != "" {
				fmt.Println("✓ [SYMULACJA] Raport miesięczny")
			}
			return
		}

		fmt.Printf("→ Łączę z drukarką %s:%d...\n", cfg.Printer.Host, cfg.Printer.Port)

		enc, err := parseEncoding(cfg.Encoding)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Błąd parsowania encoding: %v\n", err)
			os.Exit(1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Printer.Timeout)*time.Second)
		defer cancel()

		client, err := Dial(ctx, fmt.Sprintf("%s:%d", cfg.Printer.Host, cfg.Printer.Port),
			enc, time.Duration(cfg.Printer.Timeout)*time.Second,
			cfg.Printer.LogTX, cfg.Printer.LogRX)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Błąd połączenia z drukarką: %v\n", err)
			os.Exit(1)
		}
		defer client.Close()

		fc := NewFiscalClient(client, cfg.Fiscal.VATRate, cfg.Fiscal.PaymentType)
		fmt.Println("✓ Połączono z drukarką")

		if *dailyReport != "" {
			fmt.Println("→ Drukuję raport dobowy...")
			if err := fc.DailyReport(""); err != nil {
				fmt.Fprintf(os.Stderr, "❌ BŁĄD RAPORTU DOBOWEGO: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ Raport dobowy wydrukowany")
		}

		if *monthlyReport != "" {
			reportType := "pełny"
			if *monthlyReportSummary {
				reportType = "skrócony"
			}
			fmt.Printf("→ Drukuję raport miesięczny (%s)...\n", reportType)
			if err := fc.MonthlyReport(*monthlyReport, *monthlyReportSummary); err != nil {
				fmt.Fprintf(os.Stderr, "❌ BŁĄD RAPORTU MIESIĘCZNEGO: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ Raport miesięczny wydrukowany")
		}

		return
	}

	if *csvPath == "" {
		fmt.Fprintln(os.Stderr, "Błąd: wymagany parametr -csv")
		fmt.Fprintln(os.Stderr, "Użycie: druk -csv reports/01.csv [-config config.json]")
		fmt.Fprintln(os.Stderr, "lub: druk -create-config [-config config.json]")
		fmt.Fprintln(os.Stderr, "lub: druk -daily-report [YYYY-MM-DD] [-config config.json]")
		fmt.Fprintln(os.Stderr, "lub: druk -monthly-report [-config config.json]")
		os.Exit(1)
	}

	fmt.Printf("→ Wczytuję konfigurację z %s...\n", *configPath)
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Błąd wczytywania konfiguracji: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Konfiguracja wczytana")

	fmt.Printf("→ Wczytuję dane produktów z %s...\n", *dataPath)
	dataConfig, err := LoadData(*dataPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Błąd wczytywania danych: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Dane produktów wczytane")

	fmt.Printf("→ Wczytuję transakcje z %s...\n", *csvPath)
	var transactions []Transaction

	info, err := os.Stat(*csvPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Błąd dostępu do %s: %v\n", *csvPath, err)
		os.Exit(1)
	}

	if info.IsDir() {
		transactions, err = ParseCSVDirectory(*csvPath)
	} else {
		transactions, err = ParseCSVFile(*csvPath)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Błąd parsowania CSV: %v\n", err)
		os.Exit(1)
	}

	if len(transactions) == 0 {
		fmt.Fprintln(os.Stderr, "Błąd: brak transakcji w plikach CSV")
		os.Exit(1)
	}

	fmt.Printf("✓ Wczytano %d transakcji\n", len(transactions))

	grouped := GroupByDate(transactions)
	dates := GetUniqueDates(transactions)
	fmt.Printf("✓ Znaleziono %d unikalnych dni\n", len(dates))

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	var fc *FiscalClient
	if !*dryRun {
		fmt.Printf("→ Łączę z drukarką %s:%d...\n", cfg.Printer.Host, cfg.Printer.Port)

		enc, err := parseEncoding(cfg.Encoding)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Błąd parsowania encoding: %v\n", err)
			os.Exit(1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Printer.Timeout)*time.Second)
		defer cancel()

		client, err := Dial(ctx, fmt.Sprintf("%s:%d", cfg.Printer.Host, cfg.Printer.Port),
			enc, time.Duration(cfg.Printer.Timeout)*time.Second,
			cfg.Printer.LogTX, cfg.Printer.LogRX)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Błąd połączenia z drukarką: %v\n", err)
			os.Exit(1)
		}
		defer client.Close()

		fc = NewFiscalClient(client, cfg.Fiscal.VATRate, cfg.Fiscal.PaymentType)
		fmt.Println("✓ Połączono z drukarką")
	} else {
		fmt.Println("⚠ TRYB TESTOWY - symulacja bez drukarki")
	}

	totalReceipts := 0
	totalErrors := 0

	for _, date := range dates {
		dayTransactions := grouped[date]
		fmt.Printf("\n═══════════════════════════════════════\n")
		fmt.Printf("📅 Data: %s (%d paragonów)\n", date, len(dayTransactions))
		fmt.Printf("═══════════════════════════════════════\n")

		selector := NewProductSelector(cfg, dataConfig, rnd)

		for i, trans := range dayTransactions {
			receiptNum := i + 1
			fmt.Printf("\n[%d/%d] Paragon %.2f zł... ", receiptNum, len(dayTransactions), float64(trans.Amount)/100.0)

			products, err := selector.SelectProducts(trans.Amount)
			if err != nil {
				fmt.Printf("❌ BŁĄD: %v\n", err)
				totalErrors++
				continue
			}

			receipt := &Receipt{
				Total: trans.Amount,
			}

			for _, p := range products {
				receipt.Lines = append(receipt.Lines, ReceiptLine{
					Name:     p.Name,
					Price:    p.Price,
					Quantity: 1.0,
					VATRate:  cfg.Fiscal.VATRate,
				})
			}

			fmt.Println("✓")
			for _, line := range receipt.Lines {
				fmt.Printf("  • %s: %.2f zł\n", line.Name, float64(line.Price)/100.0)
			}

			if !*dryRun {
				if err := fc.PrintReceipt(receipt); err != nil {
					fmt.Printf("  ❌ BŁĄD DRUKOWANIA: %v\n", err)
					totalErrors++
					continue
				}
			}

			if err := selector.DecrementStockPermanent(products); err != nil {
				fmt.Printf("  ⚠ OSTRZEŻENIE: błąd aktualizacji stanu: %v\n", err)
			}

			totalReceipts++

			if !*dryRun {
				time.Sleep(500 * time.Millisecond)
			}
		}

		fmt.Print("\n→ Czy wydrukować raport dobowy? [t/N]: ")

		var printReport bool
		if !*dryRun {
			var response string
			fmt.Scanln(&response)
			response = strings.ToLower(strings.TrimSpace(response))
			printReport = (response == "t" || response == "tak" || response == "y" || response == "yes")

			if printReport {
				fmt.Println("→ Drukuję raport dobowy...")
				if err := fc.DailyReport(""); err != nil {
					fmt.Printf("❌ BŁĄD RAPORTU DOBOWEGO: %v\n", err)
					totalErrors++
				} else {
					fmt.Println("✓ Raport dobowy wydrukowany")
				}
				time.Sleep(2 * time.Second)
			} else {
				fmt.Println("⊘ Pominięto raport dobowy")
			}
		} else {
			fmt.Println("\n✓ [SYMULACJA] Raport dobowy (pominięty w trybie testowym)")
		}
	}

	fmt.Printf("\n→ Zapisuję zaktualizowany stan magazynowy...\n")
	if err := dataConfig.SaveData(*dataPath); err != nil {
		fmt.Printf("⚠ OSTRZEŻENIE: nie udało się zapisać stanu: %v\n", err)
	} else {
		fmt.Println("✓ Stan magazynowy zapisany")
	}

	fmt.Printf("\n═══════════════════════════════════════\n")
	fmt.Printf("📊 PODSUMOWANIE\n")
	fmt.Printf("═══════════════════════════════════════\n")
	fmt.Printf("Wydrukowanych paragonów: %d\n", totalReceipts)
	fmt.Printf("Błędów: %d\n", totalErrors)
	fmt.Printf("Dni przetworzonych: %d\n", len(dates))

	fmt.Printf("\n📦 STAN MAGAZYNOWY:\n")
	for _, p := range dataConfig.Products {
		status := "✓"
		if p.Stock == 0 {
			status = "⚠"
		} else if p.Stock < 0 {
			status = "❌"
		}
		fmt.Printf("  %s %-15s: %d szt. (użyto: %d)\n", status, p.Name, p.Stock, p.Used)
	}

	if totalErrors > 0 {
		fmt.Printf("\n⚠ Zakończono z błędami\n")
		os.Exit(1)
	}

	fmt.Printf("\n✓ Zakończono pomyślnie\n")
}
