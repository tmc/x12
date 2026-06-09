# x12

[![Go Reference](https://pkg.go.dev/badge/github.com/tmc/x12.svg)](https://pkg.go.dev/github.com/tmc/x12)

x12 is a Go library for handling EDI X12 documents.

## Features

- Decoding (`Decode`, `NewDecoder`)
- Envelope validation (`Document.Validate`)
- Encoding (`Marshal`, `NewEncoder`)

## Usage

```go
doc, err := x12.Decode(r)
if err != nil {
	// ...
}
st := doc.Interchange.FunctionGroups[0].Transactions[0].Header
fmt.Println(st.IDCode, st.ControlNumber)

out, err := x12.Marshal(doc)
```

See the [package documentation](https://pkg.go.dev/github.com/tmc/x12) for
the document model and runnable examples.

## Contributing

We welcome contributions to x12! If you find any issues or have suggestions for improvements, please feel free to open an issue or submit a pull request.

## License

x12 is released under the [MIT License](LICENSE).
