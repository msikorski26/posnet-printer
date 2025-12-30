package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Product struct {
	Name     string  `json:"name"`
	MinPrice float64 `json:"min_price"`
	MaxPrice float64 `json:"max_price"`
	Stock    int     `json:"stock"`
}

type PrinterConfig struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Timeout int    `json:"timeout"`
	LogTX   bool   `json:"log_tx"`
	LogRX   bool   `json:"log_rx"`
}

type FiscalConfig struct {
	VATRate        int `json:"vat_rate"`
	PaymentType    int `json:"payment_type"`
	ShippingChance int `json:"shipping_chance"`
	ShippingPrice  int `json:"shipping_price"`
}

type Config struct {
	Printer  PrinterConfig `json:"printer"`
	Fiscal   FiscalConfig  `json:"fiscal"`
	Products []Product     `json:"products"`
	Encoding string        `json:"encoding"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("błąd odczytu pliku config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("błąd parsowania JSON: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

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

func (c *Config) GetAvailableProducts() []Product {
	available := make([]Product, 0)
	for _, p := range c.Products {
		if p.Stock > 0 {
			available = append(available, p)
		}
	}
	return available
}

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
			VATRate:        0,
			PaymentType:    8,
			ShippingChance: 25,
			ShippingPrice:  1999,
		},
		Products: []Product{
			{Name: "Spodnie", MinPrice: 50, MaxPrice: 90, Stock: 100},
			{Name: "Sukienka", MinPrice: 90, MaxPrice: 150, Stock: 80},
			{Name: "Kombinezon", MinPrice: 150, MaxPrice: 250, Stock: 50},
			{Name: "Kurtka", MinPrice: 250, MaxPrice: 400, Stock: 40},
			{Name: "Bluzka", MinPrice: 30, MaxPrice: 60, Stock: 150},
			{Name: "Perfumy", MinPrice: 50, MaxPrice: 150, Stock: 60},
			{Name: "Majtki", MinPrice: 20, MaxPrice: 50, Stock: 200},
			{Name: "Leginsy", MinPrice: 40, MaxPrice: 60, Stock: 120},
			{Name: "Sweter", MinPrice: 90, MaxPrice: 200, Stock: 70},
			{Name: "Akcesoria kosmetyczne", MinPrice: 0, MaxPrice: 10, Stock: 228},
		},
		Encoding: "cp1250",
	}
}
