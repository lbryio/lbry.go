package main

import (
	"fmt"
	"os"

	"github.com/lbryio/lbry.go/v2/schema/claim"
)

func main() {
	args := os.Args[1:]
	if len(args) == 1 {
		claimBytes := []byte(args[0])
		decoded, err := claim.DecodeClaimBytes(claimBytes, "lbrycrd_main")
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
		claimHex := args[0]
		decoded, err := claim.DecodeClaimHex(claimHex, "lbrycrd_main")
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
