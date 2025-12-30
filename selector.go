package main

import (
	"fmt"
	"math/rand"
)

// SelectedProduct reprezentuje wylosowany produkt z ceną
type SelectedProduct struct {
	Name  string
	Price int // w groszach
}

// ProductSelector obsługuje losowanie produktów
type ProductSelector struct {
	config *Config
	rnd    *rand.Rand
}

// NewProductSelector tworzy nowy selektor produktów
func NewProductSelector(config *Config, rnd *rand.Rand) *ProductSelector {
	return &ProductSelector{
		config: config,
		rnd:    rnd,
	}
}

// SelectProducts dobiera produkty do dokładnej kwoty
// Zwraca listę produktów które sumują się dokładnie do targetAmount (w groszach)
func (ps *ProductSelector) SelectProducts(targetAmount int) ([]SelectedProduct, error) {
	// 1. Sprawdź czy dodać wysyłkę (30% szans)
	var selected []SelectedProduct
	remainingAmount := targetAmount

	if ps.rnd.Intn(100) < ps.config.Fiscal.ShippingChance {
		shippingPrice := ps.config.Fiscal.ShippingPrice
		if remainingAmount >= shippingPrice {
			selected = append(selected, SelectedProduct{
				Name:  "Wysyłka",
				Price: shippingPrice,
			})
			remainingAmount -= shippingPrice
		}
	}

	// 2. Losuj produkty dla pozostałej kwoty
	products, err := ps.findProductCombination(remainingAmount, targetAmount >= 10000) // >100zł = możliwość duplikatów
	if err != nil {
		return nil, err
	}

	selected = append(selected, products...)

	// 3. Weryfikacja sumy
	total := 0
	for _, p := range selected {
		total += p.Price
	}
	if total != targetAmount {
		return nil, fmt.Errorf("nie udało się dopasować produktów do kwoty %d gr (uzyskano %d gr)", targetAmount, total)
	}

	return selected, nil
}

// findProductCombination znajduje kombinację produktów sumującą się do targetAmount
func (ps *ProductSelector) findProductCombination(targetAmount int, allowDuplicates bool) ([]SelectedProduct, error) {
	if targetAmount <= 0 {
		return []SelectedProduct{}, nil
	}

	// Konwertujemy targetAmount z groszy na złote dla łatwiejszego dopasowania
	targetZL := float64(targetAmount) / 100.0

	// Próbujemy kilka razy z różnymi losowymi kombinacjami
	maxAttempts := 1000
	for attempt := 0; attempt < maxAttempts; attempt++ {
		result := ps.tryFindCombination(targetZL, allowDuplicates, 10) // max 10 produktów
		if result != nil {
			// Weryfikacja sumy
			total := 0
			for _, p := range result {
				total += p.Price
			}
			if total == targetAmount {
				return result, nil
			}
		}
	}

	return nil, fmt.Errorf("nie znaleziono kombinacji produktów dla kwoty %.2f zł po %d próbach", targetZL, maxAttempts)
}

// tryFindCombination próbuje znaleźć kombinację produktów (greedy + backtracking)
func (ps *ProductSelector) tryFindCombination(targetZL float64, allowDuplicates bool, maxProducts int) []SelectedProduct {
	if targetZL < 0.01 {
		return []SelectedProduct{}
	}
	if maxProducts <= 0 {
		return nil
	}

	available := ps.config.GetAvailableProducts()
	if len(available) == 0 {
		return nil
	}

	// Sortuj produkty - preferuj te które najlepiej pasują do pozostałej kwoty
	ps.shuffleProducts(available)

	for _, product := range available {
		if product.MinPrice > targetZL {
			continue
		}
		if product.Stock <= 0 {
			continue
		}

		// Oblicz optymalną cenę dla tego produktu
		var priceGr int

		// Jeśli pozostała kwota jest mała (< 1.50 zł), użyj dokładnie tyle ile zostało
		if targetZL < 1.50 && targetZL >= product.MinPrice && targetZL <= product.MaxPrice {
			priceGr = int(targetZL * 100)
		} else {
			// Preferuj wyższe ceny z zakresu (70-100% zakresu) dla większej wartości produktów
			maxPrice := product.MaxPrice
			if maxPrice > targetZL {
				maxPrice = targetZL
			}

			minGr := int(product.MinPrice * 100)
			maxGr := int(maxPrice * 100)

			if maxGr < minGr {
				continue
			}

			// Wybierz cenę z górnych 30% zakresu dla większej wartości
			rangeSize := maxGr - minGr
			lowerBound := minGr + int(float64(rangeSize)*0.7)
			if lowerBound > maxGr {
				lowerBound = minGr
			}

			if lowerBound >= maxGr {
				priceGr = maxGr
			} else {
				priceGr = lowerBound + ps.rnd.Intn(maxGr-lowerBound+1)
			}
		}

		priceZL := float64(priceGr) / 100.0
		remaining := targetZL - priceZL

		if remaining < -0.01 {
			continue
		}

		// Jeśli pozostała mała kwota (< 5 zł), spróbuj dostosować cenę bieżącego produktu
		if remaining > 0 && remaining < 5.0 {
			adjustedPriceGr := int((targetZL) * 100)
			if adjustedPriceGr >= int(product.MinPrice*100) && adjustedPriceGr <= int(product.MaxPrice*100) {
				// Możemy dostosować cenę tego produktu żeby pokryć całość
				return []SelectedProduct{{
					Name:  product.Name,
					Price: adjustedPriceGr,
				}}
			}
		}

		// Szukaj produktów na resztę
		var restProducts []SelectedProduct
		if remaining > 0.01 {
			ps.decrementStockTemporary(product.Name)
			restProducts = ps.tryFindCombination(remaining, allowDuplicates, maxProducts-1)
			ps.incrementStockTemporary(product.Name)
		} else {
			restProducts = []SelectedProduct{}
		}

		if remaining < 0.01 || restProducts != nil {
			result := []SelectedProduct{{
				Name:  product.Name,
				Price: priceGr,
			}}
			if restProducts != nil {
				result = append(result, restProducts...)
			}
			return result
		}
	}

	return nil
}

// shuffleProducts tasuje listę produktów (Fisher-Yates)
func (ps *ProductSelector) shuffleProducts(products []Product) {
	for i := len(products) - 1; i > 0; i-- {
		j := ps.rnd.Intn(i + 1)
		products[i], products[j] = products[j], products[i]
	}
}

// decrementStockTemporary tymczasowo zmniejsza stan (dla algorytmu)
func (ps *ProductSelector) decrementStockTemporary(name string) {
	for i := range ps.config.Products {
		if ps.config.Products[i].Name == name {
			ps.config.Products[i].Stock--
			return
		}
	}
}

// incrementStockTemporary przywraca stan (backtracking)
func (ps *ProductSelector) incrementStockTemporary(name string) {
	for i := range ps.config.Products {
		if ps.config.Products[i].Name == name {
			ps.config.Products[i].Stock++
			return
		}
	}
}

// DecrementStockPermanent trwale zmniejsza stan produktów
func (ps *ProductSelector) DecrementStockPermanent(products []SelectedProduct) error {
	for _, p := range products {
		if p.Name == "Wysyłka" {
			continue // wysyłka nie ma stanu
		}
		if err := ps.config.DecrementStock(p.Name); err != nil {
			return err
		}
	}
	return nil
}
