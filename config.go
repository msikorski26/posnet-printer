package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Product reprezentuje produkt w magazynie
type Product struct {
	Name     string  `json:"name"`      // Nazwa produktu na paragonie
	MinPrice float64 `json:"min_price"` // Minimalna cena w zakresie (zł)
	MaxPrice float64 `json:"max_price"` // Maksymalna cena w zakresie (zł)
	Stock    int     `json:"stock"`     // Liczba sztuk na stanie
}

// PrinterConfig zawiera ustawienia drukarki fiskalnej
type PrinterConfig struct {
	Host    string `json:"host"`    // IP drukarki
	Port    int    `json:"port"`    // Port drukarki
	Timeout int    `json:"timeout"` // Timeout w sekundach
	LogTX   bool   `json:"log_tx"`  // Logowanie wysyłanych ramek
	LogRX   bool   `json:"log_rx"`  // Logowanie odbieranych ramek
}

// FiscalConfig zawiera ustawienia fiskalne
type FiscalConfig struct {
	VATRate        int `json:"vat_rate"`        // Numer stawki VAT (0-6)
	PaymentType    int `json:"payment_type"`    // Typ płatności: 0=gotówka, 2=karta, 8=przelew
	ShippingChance int `json:"shipping_chance"` // Szansa na wysyłkę w % (np. 30)
	ShippingPrice  int `json:"shipping_price"`  // Cena wysyłki w groszach (np. 1999 = 19,99 zł)
}

// Config to główna struktura konfiguracji
type Config struct {
	Printer  PrinterConfig `json:"printer"`
	Fiscal   FiscalConfig  `json:"fiscal"`
	Products []Product     `json:"products"`
	Encoding string        `json:"encoding"` // cp1250, latin2, mazovia, ascii
}

// LoadConfig wczytuje konfigurację z pliku JSON
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("błąd odczytu pliku config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("błąd parsowania JSON: %w", err)
	}

	// Walidacja
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate sprawdza poprawność konfiguracji
func (c *Config) Validate() error {
	if c.Printer.Host == "" {
		return fmt.Errorf("brak adresu IP drukarki")
	}
	if c.Printer.Port <= 0 || c.Printer.Port > 65535 {
		return fmt.Errorf("nieprawidłowy port drukarki: %d", c.Printer.Port)
	}
	if c.Fiscal.VATRate < 0 || c.Fiscal.VATRate > 6 {
		return fmt.Errorf("nieprawidłowa stawka VAT: %d (dozwolone 0-6)", c.Fiscal.VATRate)
	}
	validPaymentTypes := map[int]bool{0: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true}
	if !validPaymentTypes[c.Fiscal.PaymentType] {
		return fmt.Errorf("nieprawidłowy typ płatności: %d", c.Fiscal.PaymentType)
	}
	if c.Fiscal.ShippingChance < 0 || c.Fiscal.ShippingChance > 100 {
		return fmt.Errorf("szansa na wysyłkę poza zakresem 0-100%%: %d", c.Fiscal.ShippingChance)
	}
	if len(c.Products) == 0 {
		return fmt.Errorf("brak produktów w konfiguracji")
	}
	for i, p := range c.Products {
		if p.Name == "" {
			return fmt.Errorf("produkt #%d: brak nazwy", i)
		}
		if p.MinPrice < 0 || p.MaxPrice < 0 {
			return fmt.Errorf("produkt %s: ujemne ceny", p.Name)
		}
		if p.MinPrice > p.MaxPrice {
			return fmt.Errorf("produkt %s: min_price > max_price", p.Name)
		}
		if p.Stock < 0 {
			return fmt.Errorf("produkt %s: ujemny stan", p.Name)
		}
	}
	return nil
}

// SaveConfig zapisuje konfigurację do pliku JSON
func (c *Config) SaveConfig(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("błąd serializacji JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("błąd zapisu pliku config: %w", err)
	}

	return nil
}

// GetAvailableProducts zwraca produkty które są dostępne na stanie
func (c *Config) GetAvailableProducts() []Product {
	available := make([]Product, 0)
	for _, p := range c.Products {
		if p.Stock > 0 {
			available = append(available, p)
		}
	}
	return available
}

// DecrementStock zmniejsza stan produktu o 1
func (c *Config) DecrementStock(productName string) error {
	for i := range c.Products {
		if c.Products[i].Name == productName {
			if c.Products[i].Stock <= 0 {
				return fmt.Errorf("produkt %s: brak na stanie", productName)
			}
			c.Products[i].Stock--
			return nil
		}
	}
	return fmt.Errorf("produkt %s: nie znaleziono", productName)
}

// CreateExampleConfig tworzy przykładową konfigurację
func CreateExampleConfig() *Config {
	return &Config{
		Printer: PrinterConfig{
			Host:    "192.168.69.45",
			Port:    12345,
			Timeout: 5,
			LogTX:   false,
			LogRX:   true,
		},
		Fiscal: FiscalConfig{
			VATRate:        0,     // 23% VAT (stawka A, vt0)
			PaymentType:    8,     // Przelew
			ShippingChance: 25,    // 25% szans
			ShippingPrice:  1999,  // 19,99 zł
		},
		Products: []Product{
			{Name: "Spodnie", MinPrice: 50, MaxPrice: 90, Stock: 100},
			{Name: "Sukienka", MinPrice: 90, MaxPrice: 150, Stock: 80},
			{Name: "Kombinezon", MinPrice: 150, MaxPrice: 250, Stock: 50},
			{Name: "Kurtka", MinPrice: 250, MaxPrice: 400, Stock: 40},
			{Name: "Bluzka", MinPrice: 0, MaxPrice: 60, Stock: 150},
			{Name: "Perfumy", MinPrice: 50, MaxPrice: 150, Stock: 60},
			{Name: "Majtki", MinPrice: 20, MaxPrice: 50, Stock: 200},
			{Name: "Leginsy", MinPrice: 40, MaxPrice: 60, Stock: 120},
			{Name: "Sweter", MinPrice: 90, MaxPrice: 200, Stock: 70},
			{Name: "Akcesoria kosmetyczne", MinPrice: 0, MaxPrice: 10, Stock: 228},
		},
		Encoding: "cp1250",
	}
}
