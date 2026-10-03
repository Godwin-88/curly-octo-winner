package sms

// MaxSMSUnits is the longest message a school may send, in billable units.
// Longer text is refused with a clear error rather than silently truncated or
// under-counted.
const MaxSMSUnits = 3

// Encoding names reported to the composer.
const (
	EncodingGSM7 = "GSM-7"
	EncodingUCS2 = "Unicode"
)

// gsm7Basic is the GSM 03.38 default alphabet: one septet each.
const gsm7Basic = "@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ !\"#¤%&'()*+,-./0123456789:;<=>?" +
	"¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà"

// gsm7Extended characters are sent as an escape plus a character: two septets.
const gsm7Extended = "\f^{}\\[~]|€"

var (
	gsm7BasicSet    = runeSet(gsm7Basic)
	gsm7ExtendedSet = runeSet(gsm7Extended)
)

func runeSet(s string) map[rune]struct{} {
	set := make(map[rune]struct{}, len(s))
	for _, r := range s {
		set[r] = struct{}{}
	}
	return set
}

// Segments describes how a message is billed.
type Segments struct {
	Units    int    // billable SMS units
	Encoding string // EncodingGSM7 or EncodingUCS2
	Length   int    // septets (GSM-7) or UTF-16 code units (Unicode)
	PerUnit  int    // capacity of each unit at this length
}

// CountSegments returns the billable units for a message.
//
// A message in the GSM-7 alphabet fits 160 characters in one unit and 153 per
// unit once it is split. Any character outside that alphabet — a curly quote
// pasted from Word, an emoji, a Kiswahili text with "ng’" typed with a smart
// apostrophe — switches the whole message to UCS-2: 70 per unit, 67 when split.
// The old count assumed 160 for everything and capped at 3, so a Unicode
// message was under-billed in the estimate by more than half.
func CountSegments(message string) Segments {
	if message == "" {
		return Segments{Encoding: EncodingGSM7, PerUnit: 160}
	}

	septets := 0
	gsm := true
	for _, r := range message {
		if _, ok := gsm7BasicSet[r]; ok {
			septets++
			continue
		}
		if _, ok := gsm7ExtendedSet[r]; ok {
			septets += 2
			continue
		}
		gsm = false
		break
	}

	if gsm {
		if septets <= 160 {
			return Segments{Units: 1, Encoding: EncodingGSM7, Length: septets, PerUnit: 160}
		}
		return Segments{Units: (septets + 152) / 153, Encoding: EncodingGSM7, Length: septets, PerUnit: 153}
	}

	// UCS-2 is counted in UTF-16 code units: characters outside the BMP (most
	// emoji) take two.
	units16 := 0
	for _, r := range message {
		if r > 0xFFFF {
			units16 += 2
		} else {
			units16++
		}
	}
	if units16 <= 70 {
		return Segments{Units: 1, Encoding: EncodingUCS2, Length: units16, PerUnit: 70}
	}
	return Segments{Units: (units16 + 66) / 67, Encoding: EncodingUCS2, Length: units16, PerUnit: 67}
}

// CalculateSMSUnits returns the number of billable units for a message.
func CalculateSMSUnits(message string) int {
	return CountSegments(message).Units
}
