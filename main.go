package lbryschema_go

import (
	"os"
	"./claim"
	"fmt"
)

func main()  {
	args := os.Args[1:]
	claim_hex := args[0]
	decoded, err := claim.DecodeClaimHex(claim_hex)
	if err != nil {
		fmt.Println("Decoding error: ", err)
		return
	}
	text, err := decoded.RenderJSON()
	if err != nil {
		fmt.Println("Decoding error: ", err)
		return
	}
	fmt.Println(text)
	return
}
