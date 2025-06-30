package entry

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://www.lidl-hellas.gr/p/"
const productsPath = "products.txt"

type product struct {
	fullName       string
	url            string
	alertThreshold int
	currentPrice   int
}

func Start() {
	// scrapeProductData()
	// return
	go func() {
		for {
			products, err := loadProducts(productsPath)
			if err != nil {
				fmt.Println("Issue with loading products: ", err)
				fmt.Println("TERMINATING!")
				os.Exit(1)
			}
			discountedProducts, err := checkDiscounts(products)
			if err != nil {
				fmt.Printf("Issues with fetching products:\n %s", err.Error())
			}
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
	if len(products) == 0 {
		fmt.Println(fmt.Sprint(time.Now().Local()) + " - No discounts found")
		return
	}
	fmt.Println(fmt.Sprint(time.Now().Local()) + " - Discounts:")
	for _, product := range products {
		fmt.Printf("%s -> %d: ", product.url, product.currentPrice)
	}
}

func checkDiscounts(products []product) ([]product, error) {
	discountedProducts := []product{}
	errs := []string{}
	for _, product := range products {
		time.Sleep(time.Second)
		client := http.Client{Timeout: 5 * time.Second}
		req, _ := http.NewRequest("GET", product.url, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/119.0 Safari/537.36")
		resp, err := client.Do(req)
		if err != nil {
			errs = append(errs, fmt.Sprintf("Failed to fetch %s for this reason: %v\n", product.url, err))
			continue
		}

		product.currentPrice, product.fullName, err = scrapeProductData(resp.Body)
		if err != nil {
			errs = append(errs, fmt.Sprintf("Failed to process url body, for this reason: %v\n", err))
			continue
		}
		if product.alertThreshold >= product.currentPrice {
			discountedProducts = append(discountedProducts, product)
		}

		resp.Body.Close()
	}
	if len(errs) > 0 {
		return discountedProducts, errors.New(strings.Join(errs, "\n"))
	}
	return discountedProducts, nil
}

func scrapeProductData(body io.Reader) (int, string, error) {
	// func scrapeProductData() (int, string, error) {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}
	// Placeholder for processing HTML response

	// f, err := os.Open("test.txt")
	// if err != nil {
	// 	return 0, "", fmt.Errorf("can't open products file: %w", err)
	// }
	// defer f.Close()

	// scanner := bufio.NewScanner(f)
	// for scanner.Scan() {
	// 	line := scanner.Text()
	// 	if strings.Contains(line, "")
	// }

	return 200, "giaourti", nil
}
