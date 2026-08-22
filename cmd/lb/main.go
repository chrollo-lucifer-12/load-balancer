package main

import (
	"errors"
	"time"
)

func unreliableService() (any, error) {

	if time.Now().Unix()%2 == 0 {

		return nil, errors.New("service failed")

	}

	return "Success!", nil

}

func main() {

}
