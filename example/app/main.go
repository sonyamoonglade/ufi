package main

import (
	"log"
	"net/url"
	"time"
)

//go:generate ufi --name=Product --out=ufi_product.go --pkg=main

/*
ufi
qf-kind (multi-value,range,exact)
qf-key

ufi: "qf-kind:...[,...];qf-key=..."
*/

type Product struct {
	SKU       uint64 `ufi:"qf-kind=range,multi-value,exact;qf-key=skus"`
	Name      string
	CreatedAt time.Time
	Age       uint
	Price     float64
}

func main() {
	const dateFrom = "2025-03-28T00:00:00.774"
	const dateTo = "2025-03-28T15:00:00.774Z"
	_ = dateFrom
	_ = dateTo

	sampleFilterURL := "https://o3.ru/products"
	_, err := url.Parse(sampleFilterURL)
	if err != nil {
		log.Fatalf("[%s] is not url: %v", sampleFilterURL, err)
	}

	_, err = ParseFilters(sampleFilterURL)
	if err != nil {
		log.Fatalf("cannot parse filter: %v", err)
	}

	/*	products := []Product{
		{Age: 5, Name: "bike"},
		{Age: 11, Name: "cycle"},
		{Age: 15, Name: "thermometer"},
		{Age: 21, Name: "laptop"},
	}*/
}
