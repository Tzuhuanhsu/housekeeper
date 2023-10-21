package main

import (
	"housekeepr/orderSys"
)

func main() {
	orderSys := new(orderSys.OrderSys)
	orderSys.Run()
}

func init() {

}
