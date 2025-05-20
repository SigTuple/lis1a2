package tests

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/therealriteshkudalkar/lis1a2"
)

func TestCalculateChecksumOfASTMConnection(t *testing.T) {
	tests := []struct {
		name     string
		expected []byte
		input    []byte
	}{
		{
			"Testing frame containing characters with accent that gave NAK",
			[]byte{55, 54},
			[]byte{2, 53, 32, 108, 105, 116, 116, 195, 169, 114, 97, 116, 117, 114, 101, 32, 108, 97, 116, 105,
				110, 101, 32, 99, 108, 97, 115, 115, 105, 113, 117, 101, 32, 100, 97, 116, 97, 110, 116, 32, 100, 101,
				32, 52, 53, 32, 97, 118, 46, 32, 74, 46, 45, 67, 46, 44, 32, 108, 101, 32, 114, 101, 110, 100, 97, 110,
				116, 32, 118, 105, 101, 117, 120, 32, 100, 101, 32, 50, 48, 48, 48, 32, 97, 110, 115, 46, 38, 88, 68,
				38, 38, 88, 65, 38, 124, 71, 13, 3, 55, 54, 13, 10},
		},
		{
			"Testing frame not containing characters with accent that gave ACK",
			[]byte{66, 53},
			[]byte{2, 52, 67, 124, 49, 124, 80, 124, 82, 101, 99, 101, 105, 118, 101, 100, 32, 99, 111, 109, 109,
				101, 110, 116, 58, 32, 38, 88, 68, 38, 38, 88, 65, 38, 38, 88, 68, 38, 38, 88, 65, 38, 85, 115, 101,
				114, 32, 99, 111, 109, 109, 101, 110, 116, 115, 58, 32, 38, 88, 68, 38, 38, 88, 65, 38, 38, 88, 68, 38,
				38, 88, 65, 38, 83, 108, 105, 100, 101, 32, 99, 111, 109, 109, 101, 110, 116, 115, 58, 32, 38, 88, 68,
				38, 38, 88, 65, 38, 82, 66, 67, 58, 32, 67, 111, 110, 116, 114, 97, 105, 114, 101, 109, 101, 110, 116,
				32, 195, 160, 32, 117, 110, 101, 32, 111, 112, 105, 110, 105, 111, 110, 32, 114, 195, 169, 112, 97, 110,
				100, 117, 101, 44, 32, 108, 101, 32, 76, 111, 114, 101, 109, 32, 73, 112, 115, 117, 109, 32, 110, 39,
				101, 115, 116, 32, 112, 97, 115, 32, 115, 105, 109, 112, 108, 101, 109, 101, 110, 116, 32, 100, 117,
				32, 116, 101, 120, 116, 101, 32, 97, 108, 195, 169, 97, 116, 111, 105, 114, 101, 46, 32, 73, 108, 32,
				116, 114, 111, 117, 118, 101, 32, 115, 101, 115, 32, 114, 97, 99, 105, 110, 101, 115, 32, 100, 97, 110,
				115, 32, 117, 110, 101, 32, 111, 101, 117, 118, 114, 101, 32, 100, 101, 32, 108, 97, 23, 66, 53, 13,
				10},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a dummy ASTMConnection Object
			astmConnObject := lis1a2.NewASTMConnection(nil, false)
			output := astmConnObject.CalculateChecksum(test.input[1 : len(test.input)-4])
			fmt.Println("String Frame:", string(test.input[1:len(test.input)-6]))
			fmt.Printf("Length: %v\n", len(test.input))
			time.Sleep(time.Second)
			if !bytes.Equal(output, test.expected) {
				t.Errorf("The expected checksum is different from calculated one. "+
					"Calculated: %v, Expected:%v\n", output, test.expected)
			}
		})
	}
}
