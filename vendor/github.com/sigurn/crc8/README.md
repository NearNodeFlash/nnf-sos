
# crc8
[![Build Status](https://travis-ci.org/sigurn/crc8.svg?branch=master)](https://travis-ci.org/sigurn/crc8)
[![Coverage Status](https://coveralls.io/repos/sigurn/crc8/badge.svg?branch=master&service=github)](https://coveralls.io/github/sigurn/crc8?branch=master)
[![GoDoc](https://godoc.org/github.com/sigurn/crc8?status.svg)](https://godoc.org/github.com/sigurn/crc8)

Go implementation of CRC-8 calculation for majority of widely-used polinomials.

## Usage
```go
package main

import (
	"fmt"
	"github.com/sigurn/crc8"
)

func main() {
	table := crc8.MakeTable(crc8.CRC8_MAXIM)
	crc := crc8.Checksum([]byte("Hello world!"), table)
	fmt.Printf("CRC-8 MAXIM: %X", crc)
}
```

## Documentation
For more documentation see [package documentation](https://godoc.org/github.com/sigurn/crc8)

## License

The MIT License (MIT)

Copyright (c) 2015 sigurn

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.


