package vyper

import (
	"fmt"
	"unicode/utf8"
)

func ConvertByteArray(byteArray []byte) string {
	var result string
	for len(byteArray) > 0 {
		r, size := utf8.DecodeRune(byteArray)
		if r == utf8.RuneError && size == 1 {
			fmt.Println("Error decoding rune.")
			break
		}
		// Add the rune to the result string
		result += string(r)
		// Move to the next rune in the byte array
		byteArray = byteArray[size:]
	}
	return result
}
