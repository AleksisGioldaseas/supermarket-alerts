package entry

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/AleksisGioldaseas/personal-lidl-discount-tracker/src/lidl"
)

const productsPath = "products.txt"

func Start() {
	go func() {
		for {
			data, err := openProductFile(productsPath)
			if err != nil {
				fmt.Println("Issue with loading products: ", err)
				fmt.Println("TERMINATING!")
				os.Exit(1)
			}

			//=====  LIDL  =================================================
			err = lidl.ReportDiscounts(data)
			if err != nil {
				fmt.Println("Issue with loading lidl products: ", err)
			}
			//==============================================================

			time.Sleep(1 * time.Hour)
		}
	}()

	// Prevent exit
	select {}
}

func openProductFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("can't open products file: %v", err)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("can't read product data: %v", data)
	}
	return string(data), nil
}
