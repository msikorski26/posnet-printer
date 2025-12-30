package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// FiscalClient rozszerza Client o operacje fiskalne
type FiscalClient struct {
	*Client
	vatRate     int
	paymentType int
}

// NewFiscalClient tworzy klienta fiskalnego
func NewFiscalClient(c *Client, vatRate, paymentType int) *FiscalClient {
	return &FiscalClient{
		Client:      c,
		vatRate:     vatRate,
		paymentType: paymentType,
	}
}

// ReceiptLine reprezentuje linię paragonu
type ReceiptLine struct {
	Name     string  // Nazwa produktu (max 80 znaków)
	Price    int     // Cena w groszach
	Quantity float64 // Ilość (domyślnie 1.0)
	VATRate  int     // Numer stawki VAT (0-6)
}

// Receipt reprezentuje cały paragon
type Receipt struct {
	Lines []ReceiptLine
	Total int // Suma w groszach
}

// DailyReport drukuje raport dobowy
func (fc *FiscalClient) DailyReport(date string) error {
	// Budujemy payload: dailyrep<TAB>da<date><TAB>
	var payload []byte
	payload = append(payload, []byte("dailyrep")...)
	payload = append(payload, TAB)

	if date != "" {
		payload = append(payload, []byte("da")...)
		payload = append(payload, []byte(date)...)
		payload = append(payload, TAB)
	}

	if err := fc.SendBytes(payload); err != nil {
		return fmt.Errorf("błąd wysyłania dailyrep: %w", err)
	}

	// Czekamy na odpowiedź
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := fc.ReadFrame(ctx)
	if err != nil {
		return fmt.Errorf("błąd odczytu odpowiedzi dailyrep: %w", err)
	}

	if strings.Contains(resp, "ERR") || strings.Contains(resp, "?") {
		return fmt.Errorf("błąd wykonania dailyrep: %s", resp)
	}

	return nil
}

// PrintReceipt drukuje paragon fiskalny
func (fc *FiscalClient) PrintReceipt(receipt *Receipt) error {
	ctx := context.Background()

	// 1. trinit - start transakcji
	if err := fc.sendTrinit(); err != nil {
		return fmt.Errorf("błąd trinit: %w", err)
	}
	if err := fc.readResponse(ctx, "trinit"); err != nil {
		return err
	}

	// 2. trline - każda linia paragonu
	for i, line := range receipt.Lines {
		if err := fc.sendTrline(line); err != nil {
			return fmt.Errorf("błąd trline #%d: %w", i, err)
		}
		if err := fc.readResponse(ctx, "trline"); err != nil {
			return err
		}
	}

	// 3. trpayment - płatność
	if err := fc.sendTrpayment(receipt.Total); err != nil {
		return fmt.Errorf("błąd trpayment: %w", err)
	}
	if err := fc.readResponse(ctx, "trpayment"); err != nil {
		return err
	}

	// 4. trend - zakończenie transakcji
	if err := fc.sendTrend(receipt.Total); err != nil {
		return fmt.Errorf("błąd trend: %w", err)
	}
	if err := fc.readResponse(ctx, "trend"); err != nil {
		return err
	}

	return nil
}

// sendTrinit wysyła komendę trinit (start transakcji)
func (fc *FiscalClient) sendTrinit() error {
	// trinit<TAB>bm0<TAB>
	var payload []byte
	payload = append(payload, []byte("trinit")...)
	payload = append(payload, TAB)
	payload = append(payload, []byte("bm0")...) // tryb online
	payload = append(payload, TAB)

	return fc.SendBytes(payload)
}

// sendTrline wysyła komendę trline (linia paragonu)
func (fc *FiscalClient) sendTrline(line ReceiptLine) error {
	// Kodowanie nazwy produktu
	nameBytes, err := encodeText(fc.enc, line.Name)
	if err != nil {
		return err
	}

	// Budujemy payload: trline<TAB>na<nazwa><TAB>vt<vat><TAB>pr<cena><TAB>il<ilość><TAB>wa<wartość><TAB>
	var payload []byte
	payload = append(payload, []byte("trline")...)
	payload = append(payload, TAB)

	// Nazwa produktu (na)
	payload = append(payload, []byte("na")...)
	payload = append(payload, nameBytes...)
	payload = append(payload, TAB)

	// Stawka VAT (vt)
	vatRate := line.VATRate
	if vatRate < 0 {
		vatRate = fc.vatRate // użyj domyślnej
	}
	payload = append(payload, []byte(fmt.Sprintf("vt%d", vatRate))...)
	payload = append(payload, TAB)

	// Cena (pr) w groszach
	payload = append(payload, []byte(fmt.Sprintf("pr%d", line.Price))...)
	payload = append(payload, TAB)

	// Ilość (il)
	qty := line.Quantity
	if qty <= 0 {
		qty = 1.0
	}
	// Format: liczba z maksymalnie 8 miejscami po przecinku
	qtyStr := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", qty), "0"), ".")
	payload = append(payload, []byte(fmt.Sprintf("il%s", qtyStr))...)
	payload = append(payload, TAB)

	// Wartość (wa) = cena * ilość
	total := int(float64(line.Price) * qty)
	payload = append(payload, []byte(fmt.Sprintf("wa%d", total))...)
	payload = append(payload, TAB)

	return fc.SendBytes(payload)
}

// sendTrpayment wysyła komendę trpayment (płatność)
func (fc *FiscalClient) sendTrpayment(amount int) error {
	// trpayment<TAB>ty<typ><TAB>wa<kwota><TAB>re0<TAB>
	var payload []byte
	payload = append(payload, []byte("trpayment")...)
	payload = append(payload, TAB)

	// Typ płatności (ty)
	payload = append(payload, []byte(fmt.Sprintf("ty%d", fc.paymentType))...)
	payload = append(payload, TAB)

	// Kwota (wa)
	payload = append(payload, []byte(fmt.Sprintf("wa%d", amount))...)
	payload = append(payload, TAB)

	// Wpłata, nie reszta (re0)
	payload = append(payload, []byte("re0")...)
	payload = append(payload, TAB)

	return fc.SendBytes(payload)
}

// sendTrend wysyła komendę trend (zakończenie transakcji)
func (fc *FiscalClient) sendTrend(total int) error {
	// trend<TAB>to<total><TAB>fp<wpłaty><TAB>re0<TAB>fe1<TAB>
	var payload []byte
	payload = append(payload, []byte("trend")...)
	payload = append(payload, TAB)

	// Total (to)
	payload = append(payload, []byte(fmt.Sprintf("to%d", total))...)
	payload = append(payload, TAB)

	// Formy płatności (fp) = total
	payload = append(payload, []byte(fmt.Sprintf("fp%d", total))...)
	payload = append(payload, TAB)

	// Reszta (re)
	payload = append(payload, []byte("re0")...)
	payload = append(payload, TAB)

	// Auto-zakończenie stopki (fe)
	payload = append(payload, []byte("fe1")...)
	payload = append(payload, TAB)

	return fc.SendBytes(payload)
}

// readResponse czyta odpowiedź i sprawdza błędy
func (fc *FiscalClient) readResponse(ctx context.Context, cmd string) error {
	readCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := fc.ReadFrame(readCtx)
	if err != nil {
		return fmt.Errorf("błąd odczytu odpowiedzi dla %s: %w", cmd, err)
	}

	// Sprawdzamy czy są błędy
	if strings.Contains(resp, "ERR") {
		return fmt.Errorf("błąd wykonania %s: %s", cmd, resp)
	}
	if strings.Contains(resp, "?") && !strings.Contains(resp, cmd) {
		// Zawiera ? ale nie zawiera nazwy komendy = błąd
		return fmt.Errorf("błąd wykonania %s: %s", cmd, resp)
	}

	return nil
}
