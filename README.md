# Gofpdf

A pure Go implementation for PDF document generation, translated from the popular FPDF PHP library (v1.86), including support to generate PDF from HTML, including images and tables.

## Installation

```bash
go get github.com/guimaraeslucas/gofpdf
```

## Features

- Core Font Support (14 standard PDF fonts)
- Images (JPEG, PNG, GIF)
- Links and Anchors
- Custom Headers and Footers
- Basic HTML rendering support
- Auto Page Breaks
- Compression support

## Basic Usage

```go
package main

import (
    "github.com/guimaraeslucas/gofpdf"
)

func main() {
    pdf := gofpdf.NewFpdf("P", "mm", "A4")
    pdf.AddPage("", "", 0)
    pdf.SetFont("Arial", "B", 16)
    pdf.Cell(40, 10, "Hello World!", 1, 0, "C", false, "")
    err := pdf.Output("F", "hello.pdf")
    if err != nil {
        panic(err)
    }
}
```

## Documentation

For full API documentation, visit [pkg.go.dev/github.com/guimaraeslucas/gofpdf](https://pkg.go.dev/github.com/guimaraeslucas/gofpdf).

## License

This project is licensed under the Mozilla Public License, v. 2.0.
Portions of this code are translated from FPDF (v1.86), originally authored by Olivier Plathey.
