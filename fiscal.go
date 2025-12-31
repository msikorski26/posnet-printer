package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type FiscalClient struct {
	*Client
	vatRate     int
	paymentType int
}

func NewFiscalClient(c *Client, vatRate, paymentType int) *FiscalClient {
	return &FiscalClient{
		Client:      c,
		vatRate:     vatRate,
		paymentType: paymentType,
	}
}

type ReceiptLine struct {
	Name     string
	Price    int
	Quantity float64
	VATRate  int
}

type Receipt struct {
	Lines []ReceiptLine
	Total int
}

func (fc *FiscalClient) DailyReport(date string) error {
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

func (fc *FiscalClient) MonthlyReport() error {
	var payload []byte
	payload = append(payload, []byte("monthrep")...)
	payload = append(payload, TAB)

	if err := fc.SendBytes(payload); err != nil {
		return fmt.Errorf("błąd wysyłania monthrep: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := fc.ReadFrame(ctx)
	if err != nil {
		return fmt.Errorf("błąd odczytu odpowiedzi monthrep: %w", err)
	}

	if strings.Contains(resp, "ERR") || strings.Contains(resp, "?") {
		return fmt.Errorf("błąd wykonania monthrep: %s", resp)
	}

	return nil
}

func (fc *FiscalClient) PrintReceipt(receipt *Receipt) error {
	ctx := context.Background()

	if err := fc.sendTrinit(); err != nil {
		return fmt.Errorf("błąd trinit: %w", err)
	}
	if err := fc.readResponse(ctx, "trinit"); err != nil {
		return err
	}

	for i, line := range receipt.Lines {
		if err := fc.sendTrline(line); err != nil {
			return fmt.Errorf("błąd trline #%d: %w", i, err)
		}
		if err := fc.readResponse(ctx, "trline"); err != nil {
			return err
		}
	}

	if err := fc.sendTrpayment(receipt.Total); err != nil {
		return fmt.Errorf("błąd trpayment: %w", err)
	}
	if err := fc.readResponse(ctx, "trpayment"); err != nil {
		return err
	}

	if err := fc.sendTrend(receipt.Total); err != nil {
		return fmt.Errorf("błąd trend: %w", err)
	}
	if err := fc.readResponse(ctx, "trend"); err != nil {
		return err
	}

	return nil
}

func (fc *FiscalClient) sendTrinit() error {
	var payload []byte
	payload = append(payload, []byte("trinit")...)
	payload = append(payload, TAB)
	payload = append(payload, []byte("bm0")...)
	payload = append(payload, TAB)

	return fc.SendBytes(payload)
}

func (fc *FiscalClient) sendTrline(line ReceiptLine) error {
	nameBytes, err := encodeText(fc.enc, line.Name)
	if err != nil {
		return err
	}

	var payload []byte
	payload = append(payload, []byte("trline")...)
	payload = append(payload, TAB)

	payload = append(payload, []byte("na")...)
	payload = append(payload, nameBytes...)
	payload = append(payload, TAB)

	vatRate := line.VATRate
	if vatRate < 0 {
		vatRate = fc.vatRate
	}
	payload = append(payload, []byte(fmt.Sprintf("vt%d", vatRate))...)
	payload = append(payload, TAB)

	payload = append(payload, []byte(fmt.Sprintf("pr%d", line.Price))...)
	payload = append(payload, TAB)

	qty := line.Quantity
	if qty <= 0 {
		qty = 1.0
	}
	qtyStr := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", qty), "0"), ".")
	payload = append(payload, []byte(fmt.Sprintf("il%s", qtyStr))...)
	payload = append(payload, TAB)

	total := int(float64(line.Price) * qty)
	payload = append(payload, []byte(fmt.Sprintf("wa%d", total))...)
	payload = append(payload, TAB)

	return fc.SendBytes(payload)
}

func (fc *FiscalClient) sendTrpayment(amount int) error {
	var payload []byte
	payload = append(payload, []byte("trpayment")...)
	payload = append(payload, TAB)

	payload = append(payload, []byte(fmt.Sprintf("ty%d", fc.paymentType))...)
	payload = append(payload, TAB)

	payload = append(payload, []byte(fmt.Sprintf("wa%d", amount))...)
	payload = append(payload, TAB)

	payload = append(payload, []byte("re0")...)
	payload = append(payload, TAB)

	return fc.SendBytes(payload)
}

func (fc *FiscalClient) sendTrend(total int) error {
	var payload []byte
	payload = append(payload, []byte("trend")...)
	payload = append(payload, TAB)

	payload = append(payload, []byte(fmt.Sprintf("to%d", total))...)
	payload = append(payload, TAB)

	payload = append(payload, []byte(fmt.Sprintf("fp%d", total))...)
	payload = append(payload, TAB)

	payload = append(payload, []byte("re0")...)
	payload = append(payload, TAB)

	payload = append(payload, []byte("fe1")...)
	payload = append(payload, TAB)

	return fc.SendBytes(payload)
}

func (fc *FiscalClient) readResponse(ctx context.Context, cmd string) error {
	readCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := fc.ReadFrame(readCtx)
	if err != nil {
		return fmt.Errorf("błąd odczytu odpowiedzi dla %s: %w", cmd, err)
	}

	if strings.Contains(resp, "ERR") {
		return fmt.Errorf("błąd wykonania %s: %s", cmd, resp)
	}
	if strings.Contains(resp, "?") && !strings.Contains(resp, cmd) {
		return fmt.Errorf("błąd wykonania %s: %s", cmd, resp)
	}

	return nil
}
