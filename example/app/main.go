package main

//go:generate ufi --name=Product --out=ufi_product.go --pkg=main

import (
	"fmt"
	"log"
	"net/url"
	"time"
)

type Product struct {
	SKU       uint64    `uti:"qf-kind=multi-value;qf-key=skus"`
	Name      string    `uti:"qf-kind=exact;qf-key=name"`
	CreatedAt time.Time `uti:"qf-kind=range,exact;qf-key=createdAt"`
	Age       uint      `uti:"qf-kind=range;qf-key=age"`
}

func main() {
	sampleFilterURL := "https://o3.ru/products?age-from=10&age-to=20"
	_, err := url.Parse(sampleFilterURL)
	if err != nil {
		log.Fatalf("[%s] is not url: %v", sampleFilterURL, err)
	}

	f, err := ParseFilters(sampleFilterURL)
	if err != nil {
		log.Fatalf("cannot parse filter: %v", err)
	}

	products := []Product{
		{Age: 5, Name: "bike"},
		{Age: 11, Name: "cycle"},
		{Age: 15, Name: "thermometer"},
		{Age: 21, Name: "laptop"},
	}

	for _, product := range products {
		if product.Age >= f.GetAgeGte() && product.Age <= f.GetAgeLte() {
			fmt.Println(product.Name)
		}
	}
}
