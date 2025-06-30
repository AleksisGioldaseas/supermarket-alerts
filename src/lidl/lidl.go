package lidl

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type apidata struct {
	Items []items `json:"items"`
}

type items struct {
	Gridbox gridbox `json:"gridbox"`
	Code    string  `json:"code"`
	Label   string  `json:"label"`
}

type gridbox struct {
	Data data `json:"data"`
}

type data struct {
	Price    price1     `json:"price"`
	LidlPlus []lidlPlus `json:"lidlPlus"`
}

type lidlPlus struct {
	Price price1 `json:"price"`
}

type price1 struct {
	Price float64 `json:"price"`
}

type product struct {
	fullName       string
	url            string
	alertThreshold int
	currentPrice   int
	code           string
	withLidlPlus   bool
}

const baseURL = "https://www.lidl-hellas.gr/q/api/search?assortment=GR&locale=el_GR&fetchsize=500&version=v2.0.0"
const linkIdentifier = "lidl-hellas.gr"

func ReportDiscounts(RawData string) error {
	lidlDataOnly := []string{}
	for _, line := range strings.Split(RawData, "\n") {
		if strings.Contains(line, linkIdentifier) {
			lidlDataOnly = append(lidlDataOnly, line)
		}
	}
	if len(lidlDataOnly) == 0 {
		return fmt.Errorf("no lidl products found")
	}

	products, err := loadProducts(lidlDataOnly)
	if err != nil {
		return fmt.Errorf("problem with loading lidl products from text: %w", err)
	}

	products, err = checkDiscounts(products, baseURL)
	if err != nil {
		fmt.Printf("issues loading some products: %v\n", err)
	}

	updateConsole(products)

	return nil
}

func scrapeProductData(product product, body io.Reader) (int, string, bool, error) {

	data, err := io.ReadAll(body)

	if err != nil {
		return 0, "", false, fmt.Errorf("failed to read request body")
	}

	errs := []string{}

	apidata := apidata{}
	err = json.Unmarshal(data, &apidata)
	if err != nil {
		errs = append(errs, fmt.Sprintf("Failed to unmarshal response from %v, err: %v\n", product.url, err))
	}

	if len(errs) > 0 {
		return 0, "", false, errors.New(strings.Join(errs, ", "))
	}

	for _, item := range apidata.Items {
		if product.code == item.Code {
			if int(item.Gridbox.Data.Price.Price*100) > 0 {
				return int(item.Gridbox.Data.Price.Price * 100), item.Label, false, nil
			}

			if len(item.Gridbox.Data.LidlPlus) > 0 {
				return int(item.Gridbox.Data.LidlPlus[0].Price.Price * 100), item.Label, true, nil
			}

			return 0, "", false, fmt.Errorf("project with products json structure")

		}
	}

	fmt.Println(apidata)

	return 0, "", false, fmt.Errorf("couldn't find item!: ")
}

func getProductLabelFromUserPage(body io.Reader) (string, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	parts := strings.Split(string(data), "<title>")
	if len(parts) != 2 {
		return "", errors.New("something wrong with first link body")
	}
	parts = strings.Split(parts[1], "</title>")
	if len(parts) != 2 {
		return "", errors.New("something wrong with first link body")
	}
	fmt.Println("found this from first link: ", parts[0])
	return parts[0], nil
}

func checkDiscounts(products []product, baseURL string) ([]product, error) {
	discountedProducts := []product{}
	errs := []string{}
	for _, product := range products {
		fmt.Println("new product: ", product)
		client := http.Client{Timeout: 3 * time.Second}

		time.Sleep(time.Second)
		fmt.Println("calling url:", product.url)
		resp, err := client.Get(product.url)

		fmt.Println("resolved")
		if err != nil {
			errs = append(errs, fmt.Sprintf("Failed to fetch %s for this reason: %v\n", product.url, err))
			continue
		}

		if resp.StatusCode != 200 {
			errs = append(errs, fmt.Sprintf("Failed to fetch %s for this reason: status -> %v\n", product.url, resp.StatusCode))
			continue
		}

		productGreekLabel, err := getProductLabelFromUserPage(resp.Body)
		resp.Body.Close()
		if err != nil {
			errs = append(errs, fmt.Sprintf("Failed to process %s body: %v\n", product.url, err))
			continue
		}

		u, err := url.Parse(baseURL)
		if err != nil {
			errs = append(errs, fmt.Sprintf("Failed to parse baseURL %s: %v\n", baseURL, err))
			continue
		}
		q := u.Query()
		q.Set("q", productGreekLabel)
		u.RawQuery = q.Encode()
		secondLink := u.String()

		client2 := http.Client{Timeout: 3 * time.Second}
		time.Sleep(time.Second)
		fmt.Println("2nd calling url:", secondLink)

		resp2, err := client2.Get(secondLink)
		fmt.Println("resolved")

		if err != nil {
			errs = append(errs, fmt.Sprintf("Failed to fetch %s for this reason: %v\n", product.url, err))
			continue
		}

		if resp.StatusCode != 200 {
			errs = append(errs, fmt.Sprintf("Failed to fetch %s for this reason: status -> %v\n", product.url, resp2.StatusCode))
			continue
		}

		product.currentPrice, product.fullName, product.withLidlPlus, err = scrapeProductData(product, resp2.Body)
		if err != nil {
			errs = append(errs, fmt.Sprintf("Failed to process url body, for this reason: %v\n", err))
			continue
		}

		if product.alertThreshold >= product.currentPrice {
			discountedProducts = append(discountedProducts, product)
		} else {
			fmt.Println("not discounted: ", product.alertThreshold, " is smaller than ", product.currentPrice)
		}

		resp2.Body.Close()
	}
	if len(errs) > 0 {
		return discountedProducts, errors.New(strings.Join(errs, "\n"))
	}
	return discountedProducts, nil
}

func loadProducts(productsText []string) ([]product, error) {
	products := []product{}

	for _, line := range productsText {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
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

		urlParts := strings.Split(parts[0], "/")
		products = append(products, product{
			url:            parts[0],
			alertThreshold: int(num * 100),
			code:           urlParts[len(urlParts)-1][1:],
		})
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
	fmt.Println("\n\n\n" + fmt.Sprint(time.Now().Local()) + " - Discounts:")
	for _, product := range products {
		extra := ""
		if product.withLidlPlus {
			extra = " with LidlPlus"
		}
		fmt.Printf("%s -> %.2f€ %s\n", product.fullName, float64(product.currentPrice)/100.0, extra)
	}
}
