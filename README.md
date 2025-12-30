# POSNET Fiscal Printer Driver

Program do automatycznego drukowania paragonów fiskalnych na drukarce POSNET przez protokół TCP.

## Funkcjonalność

- Wczytywanie transakcji z plików CSV
- Automatyczne losowanie produktów dopasowanych do kwoty paragonu
- Konfigurowalny % szans na dodanie wysyłki
- Zarządzanie stanem magazynowym
- Raport dobowy z potwierdzeniem użytkownika
- Komunikacja z drukarką fiskalną POSNET przez protokół TCP
- Tryb testowy (dry-run) bez drukarki

## Wymagania

- Go 1.16+
- Drukarka fiskalna POSNET z dostępem TCP/IP

## Instalacja

```bash
# Kompilacja
go build -o druk.exe .

# Lub uruchomienie bezpośrednio
go run . [parametry]
```

## Konfiguracja

```bash
# Utwórz przykładowy config.json
go run . -create-config

# Edytuj config.json
# - Ustaw IP i port drukarki
# - Skonfiguruj produkty i stany magazynowe
# - Dostosuj stawkę VAT i metodę płatności
```

## Użycie

```bash
# Drukowanie paragonów z pliku CSV
go run . -csv raporty/dzien.csv

# Tryb testowy (bez drukarki)
go run . -csv raporty/dzien.csv -dry-run
```

## Format pliku CSV

```
2025-12-01; 197,99
2025-12-01; 158,94
```

Format: `YYYY-MM-DD; KWOTA` (kwota z przecinkiem)

## Konfiguracja

Szczegółowa dokumentacja: [README.druk.md](README.druk.md)

## Licencja

Użytek prywatny.
