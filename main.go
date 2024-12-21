package main

import (
	"log"
	"qrcode/constants"
	"qrcode/image"
	"qrcode/qr"
)

func main() {
	version := 0
	errorCorrection := constants.ERROR_CORRECT_L
	boxSize := 30
	border := 2
	maskPattern := 5

	q, err := qr.NewQRCode(version, errorCorrection, boxSize, border, image.PilImage{}, maskPattern)
	if err != nil {
		log.Fatal(err)
	}
	q.SetVersion(4)

	data := "Hello, world!"

	if err := q.AddData(data, 0); err != nil {
		log.Fatal(err)
	}
	q.PrintASCII(
		nil,
		false,
		false,
	)
}
