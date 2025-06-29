package entry

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://www.lidl-hellas.gr/p/"
const productPath = "products.txt"

type product struct {
	fullName       string
	url            string
	alertThreshold int
}

func Start() {
	go func() {
		for {
			products, err := loadProducts()
			if err != nil {
				fmt.Println("Issue with loading products: ", err)
			}
			discountedProducts, err := checkDiscounts(products)
			updateConsole(discountedProducts)
			time.Sleep(1 * time.Hour)
		}
	}()

	// Prevent exit
	select {}
}

func loadProducts(path string) ([]product, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("can't open products file: %w", err)
	}
	defer f.Close()

	products := []product{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid product format in line: \"%s\"", line)
		}

		num, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid threshold value in line: \"%s\"", line)
		}

		products = append(products, product{
			url:            baseURL + parts[0],
			alertThreshold: int(num * 10),
		})
	}

	if scanner.Err() != nil {
		return nil, fmt.Errorf("error reading products file: %w", scanner.Err())
	}

	if len(products) == 0 {
		return nil, fmt.Errorf("no products found in file")
	}

	return products, nil
}

func updateConsole(products []product) {

}

func checkDiscounts(products []product) ([]product, error) {
	discountedProducts := []product{}
	for _, product := range products {
		time.Sleep(1)
		resp, err := http.Get(product.url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to fetch %s: %v\n", product.url, err)
			continue
		}
		resp.Body.Close()

		currentPrice, err := process(resp.Body)
		if err != nil {
			fmt.Println("Failed to process url")
		}

	}
}

func process(body io.Reader) {
	// Placeholder for processing HTML response
}
