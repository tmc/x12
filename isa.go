package x12

var interchangeIDCodeToDefinition = map[string]string{
	"01": "Duns (Dun & Bradstreet)",
	"14": "Duns Plus Suffix",
	"20": "Health Industry Number (HIN) CODE SOURCE 121: Health Industry Number",
	"27": "Carrier Identification Number as assigned by Centers for Medicare & Medicaid Services (CMS)",
	"28": "Fiscal Intermediary Identification Number as assigned by Centers for Medicare & Medicaid Services (CMS)",
	"29": "Medicare Provider and Supplier Identification Number as assigned by Centers for Medicare & Medicaid Services (CMS)",
	"30": "U.S. Federal Tax Identification Number",
	"33": "National Association of Insurance Commissioners Company Code (NAIC)",
	"ZZ": "Mutually Defined",
}

var codeToHumanMap = map[string]string{
	"ISA01_00": "No Authorization Information Present (No Meaningful Information in I02)",
	"ISA01_03": "Additional Data Identification",
	"ISA03_00": "No Security Information Present (No Meaningful Information in I04)",
	"ISA03_01": "Password",

	"ISA06_01": interchangeIDCodeToDefinition["01"],
	"ISA06_14": interchangeIDCodeToDefinition["14"],
	"ISA06_20": interchangeIDCodeToDefinition["20"],
	"ISA06_27": interchangeIDCodeToDefinition["27"],
	"ISA06_28": interchangeIDCodeToDefinition["28"],
	"ISA06_29": interchangeIDCodeToDefinition["29"],
	"ISA06_30": interchangeIDCodeToDefinition["30"],
	"ISA06_33": interchangeIDCodeToDefinition["33"],
	"ISA06_ZZ": interchangeIDCodeToDefinition["ZZ"],

	"ISA08_01": interchangeIDCodeToDefinition["01"],
	"ISA08_14": interchangeIDCodeToDefinition["14"],
	"ISA08_20": interchangeIDCodeToDefinition["20"],
	"ISA08_27": interchangeIDCodeToDefinition["27"],
	"ISA08_28": interchangeIDCodeToDefinition["28"],
	"ISA08_29": interchangeIDCodeToDefinition["29"],
	"ISA08_30": interchangeIDCodeToDefinition["30"],
	"ISA08_33": interchangeIDCodeToDefinition["33"],
	"ISA08_ZZ": interchangeIDCodeToDefinition["ZZ"],
}

// ISAElementDescription returns the human-readable description of a coded ISA
// element value. elementID is the ISA element identifier (for example "ISA06")
// and code is the value found in that element (for example "ZZ"). The boolean
// is false when no description is known for the given element and code.
func ISAElementDescription(elementID, code string) (string, bool) {
	desc, ok := codeToHumanMap[elementID+"_"+code]
	return desc, ok
}

// InterchangeIDQualifierDescription returns the human-readable definition of an
// ISA05/ISA07 interchange ID qualifier code (for example "ZZ" -> "Mutually
// Defined"). The boolean is false when the code is unknown.
func InterchangeIDQualifierDescription(code string) (string, bool) {
	desc, ok := interchangeIDCodeToDefinition[code]
	return desc, ok
}
