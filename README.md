# x12

[![Go Reference](https://pkg.go.dev/badge/github.com/tmc/x12.svg)](https://pkg.go.dev/github.com/tmc/x12)

x12 is a Go library for handling EDI x12 documents.

## Features

- Decoding
- Validation
- Encoding (Marshaling)

## Contributing

We welcome contributions to x12! If you find any issues or have suggestions for improvements, please feel free to open an issue or submit a pull request.

## License

x12 is released under the [MIT License](LICENSE).


```shell
$ go run ./examples/basics/basics.go
{
    "Interchange": {
        "Header": {
            "AuthorizationInfoQualifier": "00",
            "AuthorizationInformation": "          ",
            "SecurityInfoQualifier": "00",
            "SecurityInfo": "          ",
            "InterchangeSenderIDQualifier": "08",
            "InterchangeSenderID": "9254110060     ",
            "InterchangeReceiverIDQualifier": "ZZ",
            "InterchangeReceiverID": "123456789      ",
            "InterchangeDate": "041216",
            "InterchangeTime": "0805",
            "InterchangeControlStandardsID": "U",
            "InterchangeControlVersion": "00501",
            "InterchangeControlNumber": "000095071",
            "AcknowledgmentRequested": "0",
            "UsageIndicator": "P",
            "ComponentElementSeparator": "\u003e"
        },
        "FunctionGroups": [
            {
                "Header": {
                    "FunctionalIDCode": "AG",
                    "ApplicationSenderCode": "5137624388",
                    "ApplicationReceiverCode": "123456789",
                    "Date": "20041216",
                    "Time": "0805",
                    "GroupControlNumber": "95071",
                    "ResponsibleAgencyCode": "X",
                    "VersionReleaseIndustryID": "005010"
                },
                "Transactions": [
                    {
                        "Header": {
                            "TransactionSetIDCode": "824",
                            "TransactionSetControlNumber": "021390001"
                        },
                        "Segments": [
                            {
                                "ID": "BGN",
                                "Elements": [
                                    {
                                        "ID": "01",
                                        "Value": "11"
                                    },
                                    {
                                        "ID": "02",
                                        "Value": "FFA.ABCDEF.123456"
                                    },
                                    {
                                        "ID": "03",
                                        "Value": "20020709"
                                    },
                                    {
                                        "ID": "04",
                                        "Value": "0932"
                                    },
                                    {
                                        "ID": "05",
                                        "Value": ""
                                    },
                                    {
                                        "ID": "06",
                                        "Value": "123456789"
                                    },
                                    {
                                        "ID": "07",
                                        "Value": ""
                                    },
                                    {
                                        "ID": "08",
                                        "Value": "WQ"
                                    }
                                ]
                            },
                            {
                                "ID": "N1",
                                "Elements": [
                                    {
                                        "ID": "01",
                                        "Value": "41"
                                    },
                                    {
                                        "ID": "02",
                                        "Value": "ABC INSURANCE"
                                    },
                                    {
                                        "ID": "03",
                                        "Value": "46"
                                    },
                                    {
                                        "ID": "04",
                                        "Value": "111111111"
                                    }
                                ]
                            },
                            {
                                "ID": "PER",
                                "Elements": [
                                    {
                                        "ID": "01",
                                        "Value": "IC"
                                    },
                                    {
                                        "ID": "02",
                                        "Value": "JOHN JOHNSON"
                                    },
                                    {
                                        "ID": "03",
                                        "Value": "TE"
                                    },
                                    {
                                        "ID": "04",
                                        "Value": "8005551212"
                                    },
                                    {
                                        "ID": "05",
                                        "Value": "EX"
                                    },
                                    {
                                        "ID": "06",
                                        "Value": "1439"
                                    }
                                ]
                            },
                            {
                                "ID": "N1",
                                "Elements": [
                                    {
                                        "ID": "01",
                                        "Value": "40"
                                    },
                                    {
                                        "ID": "02",
                                        "Value": "SMITHCO"
                                    },
                                    {
                                        "ID": "03",
                                        "Value": "46"
                                    },
                                    {
                                        "ID": "04",
                                        "Value": "A1234"
                                    }
                                ]
                            },
                            {
                                "ID": "OTI",
                                "Elements": [
                                    {
                                        "ID": "01",
                                        "Value": "TA"
                                    },
                                    {
                                        "ID": "02",
                                        "Value": "TN"
                                    },
                                    {
                                        "ID": "03",
                                        "Value": "NA"
                                    },
                                    {
                                        "ID": "04",
                                        "Value": ""
                                    },
                                    {
                                        "ID": "05",
                                        "Value": ""
                                    },
                                    {
                                        "ID": "06",
                                        "Value": "20020709"
                                    },
                                    {
                                        "ID": "07",
                                        "Value": "0902"
                                    },
                                    {
                                        "ID": "08",
                                        "Value": "2"
                                    },
                                    {
                                        "ID": "09",
                                        "Value": "0001"
                                    },
                                    {
                                        "ID": "10",
                                        "Value": "834"
                                    },
                                    {
                                        "ID": "11",
                                        "Value": "005010X220A1"
                                    }
                                ]
                            }
                        ],
                        "Trailer": {
                            "NumberOfIncludedSegments": "7",
                            "TransactionSetControlNumber": "021390001"
                        }
                    }
                ],
                "Trailer": {
                    "NumberOfIncludedTransactionSets": "1",
                    "GroupControlNumber": "95071"
                }
            }
        ],
        "Trailer": {
            "NumberOfIncludedFunctionalGroups": "1",
            "InterchangeControlNumber": "000095071"
        }
    }
}
```
