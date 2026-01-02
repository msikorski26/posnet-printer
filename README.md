# POSNET Fiscal Printer Driver

Program do automatycznego drukowania paragonów fiskalnych na drukarce POSNET przez protokół TCP.

## Szybki start

```bash
# 1. Kompilacja
go build -o posnet-printer.exe

# 2. Utworzenie konfiguracji
posnet-printer.exe -create-config

# 3. Edycja config.json i data.json
#    - Ustaw IP i port drukarki w config.json
#    - Dodaj produkty w data.json

# 4. Drukowanie paragonów
posnet-printer.exe -csv reports/
```

## Spis komend

### Podstawowe komendy

```bash
# Utworzenie przykładowych plików konfiguracji
posnet-printer.exe -create-config

# Drukowanie paragonów z pojedynczego pliku CSV
posnet-printer.exe -csv reports/01.csv

# Drukowanie paragonów z całego katalogu
posnet-printer.exe -csv reports/

# Tryb testowy (bez drukarki)
posnet-printer.exe -csv reports/ -dry-run
```

### Raporty fiskalne

```bash
# Raport dobowy (zawsze dla bieżącego dnia)
posnet-printer.exe -daily-report true

# Raport miesięczny (pełny) dla bieżącego miesiąca
posnet-printer.exe -monthly-report true

# Raport miesięczny dla czerwca 2021
posnet-printer.exe -monthly-report "2021-06-19"

# Raport miesięczny skrócony
posnet-printer.exe -monthly-report true -monthly-report-summary

# Raport miesięczny skrócony dla czerwca 2021
posnet-printer.exe -monthly-report "2021-06-19" -monthly-report-summary
```

### Niestandardowa konfiguracja

```bash
# Własne ścieżki do plików konfiguracji
posnet-printer.exe -csv reports/ -config my-config.json -data my-data.json
```

## Parametry CLI

| Parametr | Typ | Opis |
|----------|-----|------|
| `-config` | string | Ścieżka do pliku konfiguracji (domyślnie: `config.json`) |
| `-data` | string | Ścieżka do pliku danych produktów (domyślnie: `data.json`) |
| `-csv` | string | Ścieżka do pliku/katalogu CSV |
| `-create-config` | bool | Utwórz przykładowe pliki config.json i data.json |
| `-dry-run` | bool | Tryb testowy bez drukarki |
| `-daily-report` | string | Wydrukuj raport dobowy (zawsze dla bieżącego dnia) |
| `-monthly-report` | string | Wydrukuj raport miesięczny (format: YYYY-MM-DD lub puste dla bieżącego miesiąca) |
| `-monthly-report-summary` | bool | Raport miesięczny w wersji skróconej |

## Format pliku CSV

```csv
2025-12-01; 197,99
2025-12-01; 158,94
2025-12-02; 230,50
```

Format: `YYYY-MM-DD; KWOTA` (kwota z przecinkiem)

## Pliki konfiguracyjne

### config.json
```json
{
  "printer": {
    "host": "192.168.1.100",
    "port": 12345,
    "timeout": 5,
    "log_tx": false,
    "log_rx": true
  },
  "fiscal": {
    "vat_rate": 0,
    "payment_type": 8,
    "shipping_chance": 25,
    "shipping_price": 1999
  },
  "encoding": "cp1250"
}
```

### data.json
```json
{
  "products": [
    {
      "name": "Produkt 1",
      "min_price": 50,
      "max_price": 90,
      "stock": 100,
      "used": 0
    }
  ]
}
```

## Funkcjonalność

- Wczytywanie transakcji z plików CSV lub katalogów
- Automatyczne losowanie produktów dopasowanych do kwoty
- Zarządzanie stanem magazynowym
- Automatyczne pytanie o raport dzienny po każdym dniu
- Manualne drukowanie raportów dobowych i miesięcznych
- Tryb testowy (dry-run)

## Wymagania

- Go 1.21+
- Drukarka fiskalna POSNET z dostępem TCP/IP
