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
