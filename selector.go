package main

import (
	"fmt"
	"math/rand"
)

type SelectedProduct struct {
	Name  string
	Price int
}

type ProductSelector struct {
	config *Config
	data   *DataConfig
	rnd    *rand.Rand
}

func NewProductSelector(config *Config, data *DataConfig, rnd *rand.Rand) *ProductSelector {
	return &ProductSelector{
		config: config,
		data:   data,
		rnd:    rnd,
	}
}

func (ps *ProductSelector) SelectProducts(targetAmount int) ([]SelectedProduct, error) {
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

	products, err := ps.findProductCombination(remainingAmount, targetAmount >= 10000)
	if err != nil {
		return nil, err
	}

	selected = append(selected, products...)

	total := 0
	for _, p := range selected {
		total += p.Price
	}
	if total != targetAmount {
		return nil, fmt.Errorf("nie udało się dopasować produktów do kwoty %d gr (uzyskano %d gr)", targetAmount, total)
	}

	return selected, nil
}

func (ps *ProductSelector) findProductCombination(targetAmount int, allowDuplicates bool) ([]SelectedProduct, error) {
	if targetAmount <= 0 {
		return []SelectedProduct{}, nil
	}

	targetZL := float64(targetAmount) / 100.0

	maxAttempts := 1000
	for attempt := 0; attempt < maxAttempts; attempt++ {
		result := ps.tryFindCombination(targetZL, allowDuplicates, 10)
		if result != nil {
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

func (ps *ProductSelector) tryFindCombination(targetZL float64, allowDuplicates bool, maxProducts int) []SelectedProduct {
	if targetZL < 0.01 {
		return []SelectedProduct{}
	}
	if maxProducts <= 0 {
		return nil
	}

	available := ps.data.GetAvailableProducts()
	if len(available) == 0 {
		return nil
	}

	ps.shuffleProducts(available)

	for _, product := range available {
		if product.MinPrice > targetZL {
			continue
		}
		if product.Stock <= 0 {
			continue
		}

		var priceGr int

		if targetZL < 1.50 && targetZL >= product.MinPrice && targetZL <= product.MaxPrice {
			priceGr = int(targetZL * 100)
		} else {
			maxPrice := product.MaxPrice
			if maxPrice > targetZL {
				maxPrice = targetZL
			}

			minGr := int(product.MinPrice * 100)
			maxGr := int(maxPrice * 100)

			if maxGr < minGr {
				continue
			}

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

		if remaining > 0 && remaining < 5.0 {
			adjustedPriceGr := int((targetZL) * 100)
			if adjustedPriceGr >= int(product.MinPrice*100) && adjustedPriceGr <= int(product.MaxPrice*100) {
				return []SelectedProduct{{
					Name:  product.Name,
					Price: adjustedPriceGr,
				}}
			}
		}

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

func (ps *ProductSelector) shuffleProducts(products []Product) {
	for i := len(products) - 1; i > 0; i-- {
		j := ps.rnd.Intn(i + 1)
		products[i], products[j] = products[j], products[i]
	}
}

func (ps *ProductSelector) decrementStockTemporary(name string) {
	for i := range ps.data.Products {
		if ps.data.Products[i].Name == name {
			ps.data.Products[i].Stock--
			return
		}
	}
}

func (ps *ProductSelector) incrementStockTemporary(name string) {
	for i := range ps.data.Products {
		if ps.data.Products[i].Name == name {
			ps.data.Products[i].Stock++
			return
		}
	}
}

func (ps *ProductSelector) DecrementStockPermanent(products []SelectedProduct) error {
	for _, p := range products {
		if p.Name == "Wysyłka" {
			continue
		}
		if err := ps.data.DecrementStock(p.Name); err != nil {
			return err
		}
	}
	return nil
}
