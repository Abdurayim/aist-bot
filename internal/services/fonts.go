package services

import _ "embed"

// The DejaVu faces are embedded so the binary can be copied anywhere and
// still render Latin, Uzbek and Cyrillic text in the generated documents.
var (
	//go:embed assets/DejaVuSans.ttf
	fontRegular []byte

	//go:embed assets/DejaVuSans-Bold.ttf
	fontBold []byte
)
