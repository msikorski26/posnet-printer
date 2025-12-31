# POSNET Fiscal Printer Driver

Program do automatycznego drukowania paragonów fiskalnych na drukarce POSNET przez protokół TCP.

## Funkcjonalność

- Wczytywanie transakcji z plików CSV lub całych katalogów
- Automatyczne losowanie produktów dopasowanych do kwoty paragonu
- Konfigurowalny % szans na dodanie wysyłki
- Zarządzanie stanem magazynowym z śledzeniem użycia
- **Automatyczne pytanie o raport dzienny po każdym dniu**
- **Manualne drukowanie raportów dobowych i miesięcznych**
- **Oddzielna konfiguracja (config.json) i dane produktów (data.json)**
- Komunikacja z drukarką fiskalną POSNET przez protokół TCP
- Tryb testowy (dry-run) bez drukarki

## Wymagania

- Go 1.21+
- Drukarka fiskalna POSNET z dostępem TCP/IP

## Instalacja

### Kompilacja ze źródeł

```bash
# Kompilacja
go build -o druk.exe .

# Lub uruchomienie bezpośrednio
go run . [parametry]
```


## Konfiguracja

```bash
# Utwórz przykładowe pliki config.json i data.json
druk -create-config

# Edytuj config.json
# - Ustaw IP i port drukarki
# - Skonfiguruj stawkę VAT i metodę płatności
# - Ustaw encoding (domyślnie cp1250)

# Edytuj data.json
# - Dodaj/edytuj produkty
# - Ustaw ceny min/max dla każdego produktu
# - Ustaw stany magazynowe
```

## Użycie

### Drukowanie paragonów

```bash
# Drukowanie z pojedynczego pliku CSV
druk -csv reports/01.csv

# Drukowanie z całego katalogu (wszystkie pliki *.csv)
druk -csv reports/

# Tryb testowy (bez drukarki)
druk -csv reports/ -dry-run

# Własna ścieżka do plików konfiguracji
druk -csv reports/ -config my-config.json -data my-data.json
```

### Manualne raporty

```bash
# Raport dobowy za konkretną datę
druk -daily-report 2024-12-31

# Raport miesięczny
druk -monthly-report

# Kombinacja raportów
druk -daily-report 2024-12-31 -monthly-report
```

## Format pliku CSV

```csv
2025-12-01; 197,99
2025-12-01; 158,94
2025-12-02; 230,50
2025-12-02; 189,00
```

Format: `YYYY-MM-DD; KWOTA` (kwota z przecinkiem)

## Automatyczne pytanie o raporty dzienne

Program automatycznie pyta o raport dzienny po zakończeniu drukowania paragonów z każdego dnia:

```
📅 Data: 2024-12-01 (10 paragonów)
...
[drukowanie paragonów]
...

→ Czy wydrukować raport dobowy za 2024-12-01? [t/N]: t
→ Drukuję raport dobowy za 2024-12-01...
✓ Raport dobowy wydrukowany
```

## Struktura plików konfiguracyjnych

### config.json
Zawiera ustawienia drukarki i konfigurację fiskalną:
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
Zawiera dane produktów i stany magazynowe:
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

Pole `used` automatycznie śledzi ile sztuk danego produktu zostało użytych.

## Parametry CLI

| Parametr | Opis | Domyślna wartość |
|----------|------|------------------|
| `-config` | Ścieżka do pliku konfiguracji | `config.json` |
| `-data` | Ścieżka do pliku danych produktów | `data.json` |
| `-csv` | Ścieżka do pliku/katalogu CSV | - |
| `-create-config` | Utwórz przykładowe pliki konfiguracji | - |
| `-dry-run` | Tryb testowy bez drukarki | `false` |
| `-daily-report` | Wydrukuj raport dobowy (format: YYYY-MM-DD) | - |
| `-monthly-report` | Wydrukuj raport miesięczny | `false` |

## Przykłady

### Podstawowe użycie
```bash
# Pierwszy raz - utworzenie konfiguracji
druk -create-config

# Drukowanie paragonów z katalogu
druk -csv reports/

# Raport dzienny
druk -daily-report 2024-12-31
```

### Zaawansowane użycie
```bash
# Drukowanie z niestandardowymi plikami konfiguracji
druk -csv december/ -config config-december.json -data data-december.json

# Testowanie bez drukarki
druk -csv reports/ -dry-run

# Manualne raporty
druk -daily-report 2024-12-31 -monthly-report
```

## Stan magazynowy

Po zakończeniu drukowania program wyświetla raport stanu magazynowego:

```
📦 STAN MAGAZYNOWY:
  ✓ Spodnie        : 85 szt. (użyto: 15)
  ✓ Sukienka       : 70 szt. (użyto: 10)
  ⚠ Kurtka         : 0 szt. (użyto: 40)
  ✓ Bluzka         : 145 szt. (użyto: 5)
```

- ✓ = dostępne na stanie
- ⚠ = brak na stanie (0)
- ❌ = ujemny stan (błąd)

## Licencja

Użytek prywatny.
