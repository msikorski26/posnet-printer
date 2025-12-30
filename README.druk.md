# POSNET Druk - Generator Paragonów Fiskalnych

Program do automatycznego drukowania paragonów fiskalnych na podstawie danych z plików CSV.

## Funkcjonalność

- Wczytywanie transakcji z plików CSV
- Automatyczne losowanie produktów dopasowanych do kwoty paragonu
- 30% szans na dodanie wysyłki (19,99 zł)
- Zarządzanie stanem magazynowym
- Automatyczne raportydzie dobowe po każdym dniu
- Komunikacja z drukarką fiskalną POSNET przez protokół TCP

## Instalacja

```bash
# Kompilacja programu
go build -o druk.exe .

# Lub uruchomienie bezpośrednio
go run . [parametry]
```

## Konfiguracja

### 1. Utworzenie pliku konfiguracji

```bash
# Utwórz przykładowy config.json
go run . -create-config -config config.json
```

### 2. Edycja konfiguracji

Plik `config.json` zawiera:

```json
{
  "printer": {
    "host": "192.168.69.45",     // IP drukarki
    "port": 12345,               // Port drukarki
    "timeout": 5,                // Timeout w sekundach
    "log_tx": false,             // Logowanie wysyłanych ramek
    "log_rx": true               // Logowanie odbieranych ramek
  },
  "fiscal": {
    "vat_rate": 2,               // Stawka VAT (0-6, zwykle 2=23%)
    "payment_type": 8,           // Typ płatności (8=przelew)
    "shipping_chance": 30,       // Szansa na wysyłkę w %
    "shipping_price": 1999       // Cena wysyłki w groszach (19,99 zł)
  },
  "products": [
    {
      "name": "Spodnie",         // Nazwa produktu
      "min_price": 50,           // Min cena (zł)
      "max_price": 90,           // Max cena (zł)
      "stock": 100               // Stan magazynowy
    },
    ...
  ],
  "encoding": "cp1250"           // Kodowanie (cp1250/latin2/mazovia/ascii)
}
```

### 3. Typy płatności

- `0` - Gotówka
- `2` - Karta
- `3` - Czek
- `4` - Bon
- `5` - Kredyt
- `6` - Inna
- `7` - Voucher
- `8` - Przelew (domyślnie)

### 4. Stawki VAT

Zależą od konfiguracji drukarki, typowo:
- `0` - PTU A (zwykle 23%)
- `1` - PTU B (zwykle 8%)
- `2` - PTU C (zwykle 5%)
- `3` - PTU D (zwykle 0%)
- `4` - Zwolnione
- `5` - Nie podlega
- `6` - Inne

## Format pliku CSV

Pliki CSV powinny być w formacie:

```
2025-12-01; 197,99
2025-12-01; 158,94
2025-12-01; 151,99
```

Format linii: `RRRR-MM-DD; KWOTA`

- Data w formacie ISO (YYYY-MM-DD)
- Separator: średnik i spacja
- Kwota: liczba dziesiętna z przecinkiem

## Użycie

### Podstawowe użycie

```bash
# Drukowanie paragonów z jednego pliku
go run . -csv raporty/01.csv -config config.json

# Drukowanie paragonów z całego katalogu
go run . -csv raporty/ -config config.json
```

### Tryb testowy (bez drukarki)

```bash
# Symulacja bez łączenia z drukarką
go run . -csv raporty/01.csv -dry-run
```

## Działanie programu

1. **Wczytanie konfiguracji** - odczyt config.json
2. **Parsowanie CSV** - wczytanie transakcji
3. **Grupowanie po datach** - pogrupowanie paragonów
4. **Dla każdego dnia:**
   - **Dla każdej transakcji:**
     - Losowanie czy dodać wysyłkę (30% szans)
     - Losowanie produktów dopasowanych do kwoty
     - Drukowanie paragonu fiskalnego
     - Aktualizacja stanu magazynowego
   - **Raport dobowy** - wydruk raportu za dzień
5. **Zapis stanu** - aktualizacja config.json

## Algorytm doboru produktów

Program automatycznie dobiera produkty tak, aby suma była **dokładnie** równa kwocie z CSV:

1. **Wysyłka (30%)**: Jeśli wylosowano, dodaj "Wysyłka 19,99 zł", pozostała kwota -= 19,99
2. **Losowanie produktów**:
   - Losuje cenę dla każdego produktu z jego zakresu (min_price - max_price)
   - Szuka kombinacji produktów sumujących się dokładnie do kwoty
   - Dla kwot >100 zł może dodać 2x ten sam produkt
   - Sprawdza stan magazynowy przed dodaniem
   - Pomija produkty bez stanu i szuka alternatyw

## Stan magazynowy

- Stan jest przechowywany w `config.json`
- Automatycznie zmniejszany po każdym paragonie
- Zapisywany po zakończeniu programu
- Produkty ze stanem 0 są pomijane przy losowaniu

## Przykład uruchomienia

```bash
# 1. Utwórz config
go run . -create-config

# 2. Edytuj config.json (dostosuj IP drukarki, produkty, stany)

# 3. Przetestuj w trybie dry-run
go run . -csv raporty/01.csv -dry-run

# 4. Uruchom produkcyjnie
go run . -csv raporty/01.csv
```

## Wyjście programu

Program wyświetla na bieżąco:
- Status wczytywania konfiguracji i CSV
- Liczbę transakcji i dni
- Dla każdego paragonu:
  - Numer paragonu i kwotę
  - Wylosowane produkty i ich ceny
  - Status drukowania
- Raport dobowy
- Podsumowanie:
  - Liczbę wydrukowanych paragonów
  - Liczbę błędów
  - Aktualny stan magazynowy

Przykład:

```
→ Wczytuję konfigurację z config.json...
✓ Konfiguracja wczytana
→ Wczytuję transakcje z raporty/01.csv...
✓ Wczytano 16 transakcji
✓ Znaleziono 1 unikalnych dni
→ Łączę z drukarką 192.168.69.45:12345...
✓ Połączono z drukarką

═══════════════════════════════════════
📅 Data: 2025-12-01 (16 paragonów)
═══════════════════════════════════════

[1/16] Paragon 197.99 zł... ✓
  • Sweter: 178.00 zł
  • Wysyłka: 19.99 zł

[2/16] Paragon 158.94 zł... ✓
  • Sukienka: 138.95 zł
  • Wysyłka: 19.99 zł

...

→ Drukuję raport dobowy za 2025-12-01...
✓ Raport dobowy wydrukowany

→ Zapisuję zaktualizowany stan magazynowy...
✓ Stan magazynowy zapisany

═══════════════════════════════════════
📊 PODSUMOWANIE
═══════════════════════════════════════
Wydrukowanych paragonów: 16
Błędów: 0
Dni przetworzonych: 1

📦 STAN MAGAZYNOWY:
  ✓ Spodnie        : 98 szt.
  ✓ Sukienka       : 76 szt.
  ✓ Kombinezon     : 50 szt.
  ✓ Kurtka         : 40 szt.
  ✓ Bluzka         : 148 szt.
  ✓ Perfumy        : 60 szt.
  ✓ Majtki         : 196 szt.
  ✓ Leginsy        : 118 szt.
  ✓ Sweter         : 68 szt.

✓ Zakończono pomyślnie
```

## Obsługa błędów

Program jest odporny na błędy:
- Nieprawidłowe linie w CSV są pomijane z ostrzeżeniem
- Błąd drukowania pojedynczego paragonu nie przerywa całego procesu
- Stan magazynowy jest przywracany (rollback) w przypadku błędu druku
- Brak produktów pasujących do kwoty jest raportowany
- Problemy z połączeniem TCP są wyświetlane ze szczegółami

## Wymagania

- Go 1.16+
- Dostęp sieciowy do drukarki POSNET
- Drukarka skonfigurowana w trybie fiskalnym
- Poprawnie skonfigurowany plik config.json

## Rozwiązywanie problemów

### Nie można połączyć z drukarką

- Sprawdź IP i port w config.json
- Upewnij się że drukarka jest włączona i podłączona do sieci
- Sprawdź czy firewall nie blokuje połączenia

### Błąd "nie znaleziono kombinacji produktów"

- Sprawdź czy zakresy cen produktów pokrywają kwoty z CSV
- Dodaj produkty o niższych/wyższych cenach
- Sprawdź czy produkty mają stan > 0
- Zwiększ liczbę dostępnych produktów

### Stan magazynowy jest ujemny

- Program pozwala na kontynuację nawet przy braku stanu
- Uzupełnij stany w config.json przed kolejnym uruchomieniem

### Błąd CRC

- Sprawdź kodowanie w config.json (encoding)
- Spróbuj użyć "ascii" jeśli są problemy z polskimi znakami

## Licencja

Program stworzony do użytku wewnętrznego.
