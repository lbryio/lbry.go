package main

import (
	"os"
	"../claim"
	"fmt"
)

func main()  {
	args := os.Args[1:]
	if len(args) == 1 {
		claim_bytes := []byte(args[0])
		decoded, err := claim.DecodeClaimBytes(claim_bytes)
		if err != nil {
			fmt.Println("Decoding error:", err)
			return
		}
		text, err := decoded.RenderJSON()
		if err != nil {
			fmt.Println("Decoding error:", err)
			return
		}
		fmt.Println(text)
		return
	} else if (len(args) == 2) && (args[1] == "--decode_hex") {
		claim_hex := args[0]
		decoded, err := claim.DecodeClaimHex(claim_hex)
		if err != nil {
			fmt.Println("Decoding error:", err)
			return
		}
		text, err := decoded.RenderJSON()
		if err != nil {
			fmt.Println("Decoding error:", err)
			return
		}
		fmt.Println(text)
		return
	} else {
		fmt.Println("encountered an error\nusage: \n\tlbryschema-cli <value to decode> [--decode_hex]")
		return
	}
}
