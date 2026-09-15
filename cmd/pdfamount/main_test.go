package main

import "testing"

func TestExtractAmount(t *testing.T) {
	tests := []struct {
		name, text, want string
	}{
		{
			name: "rechnungsbetrag de format",
			text: "Rechnungsbetrag: 1.234,56 €",
			want: "1234.56",
		},
		{
			name: "rechnungsbetrag en thousands and dot",
			text: "Rechnungsbetrag: 1,234.56",
			want: "1234.56",
		},
		{
			name: "rechnungsbetrag plain dot",
			text: "Rechnungsbetrag: 1234.56",
			want: "1234.56",
		},
		{
			name: "rechnungsbetrag de comma only",
			text: "Rechnungsbetrag: 1234,56",
			want: "1234.56",
		},
		{
			name: "currency suffix eur",
			text: "Rechnungsbetrag: 1.234,56 EUR",
			want: "1234.56",
		},
		{
			name: "zu zahlender betrag wins over rechnungsbetrag",
			text: "Zu zahlender Betrag: 10,00\nRechnungsbetrag: 99,00",
			want: "10.00",
		},
		{
			name: "rechnungsbetrag wins over endbetrag",
			text: "Endbetrag: 20,00\nRechnungsbetrag: 30,00",
			want: "30.00",
		},
		{
			name: "bruttobetrag wins over bare betrag",
			text: "Bruttobetrag: 50,00\nBetrag: 10,00",
			want: "50.00",
		},
		{
			name: "gesamtbetrag wins over bare betrag",
			text: "Gesamtbetrag: 80,00\nBetrag: 10,00",
			want: "80.00",
		},
		{
			name: "last occurrence of same label wins",
			text: "Rechnungsbetrag: 10,00\nRechnungsbetrag: 20,00",
			want: "20.00",
		},
		{
			name: "endbetrag fallback",
			text: "Endbetrag: 42,00",
			want: "42.00",
		},
		{
			name: "zahlbetrag fallback without colon",
			text: "Zahlbetrag 7,50 €",
			want: "7.50",
		},
		{
			name: "rechnungsendbetrag beats endbetrag",
			text: "Rechnungsendbetrag: 12,00\nEndbetrag: 13,00",
			want: "12.00",
		},
		{
			name: "dkv style total line",
			text: "Kundenbezogene Daten\n» TOTAL: 123,45 100,00 23,45 123,45\n",
			want: "123.45",
		},
		{
			name: "dkv gesamtsummenaufstellung grand total after marker",
			text: "» TOTAL: 111,11 100,00 11,11 111,11\n" +
				"» TOTAL: 222,22 200,00 22,22 222,22\n" +
				"Gesamtsummenaufstellung\n" +
				"Netto 240,00\n" +
				"MwSt 47,25\n" +
				"» 287,25\n",
			want: "287.25",
		},
		{
			name: "dkv gesamtsummenaufstellung total on next line",
			text: "» TOTAL: 111,11\nGesamtsummenaufstellung\n»\n287,25\n",
			want: "287.25",
		},
		{
			name: "tax rate with percent sign is not an amount",
			text: "Betrag: 19,00 % MwSt",
			want: "",
		},
		{
			name: "mehrwertsteuer rate is not an amount",
			text: "Gesamtsumme: 19,00% MwSt",
			want: "",
		},
		{
			name: "steuer word on the number line rejects it",
			text: "Betrag: 2,83 Steuer",
			want: "",
		},
		{
			name: "rejected primary falls back to a usable label",
			text: "Gesamtsumme: 19,00 % MwSt\nEndbetrag: 42,00",
			want: "42.00",
		},
		{
			name: "labeled zu zahlender betrag beats unlabeled larger number",
			text: "unlabeled 999,99\nZu zahlender Betrag: 10,00",
			want: "10.00",
		},
		{
			name: "labeled zu zahlender betrag beats lower label larger number",
			text: "Endbetrag: 999,99\nZu zahlender Betrag: 10,00",
			want: "10.00",
		},
		{
			name: "diashop style gesamtsumme with comment",
			text: "Zwischensumme\n12,34 €\nZwischensumme\n12,34 €\nVersand & Bearbeitung\n4,95 €\nGesamtsumme (inkl. Steuern)\n17,29 €\n",
			want: "17.29",
		},
		{
			name: "diashop picks inclusive total last",
			text: "Gesamtsumme (exkl. Steuern)\n12,34 €\nGesamtsumme (inkl. Steuern)\n17,29 €",
			want: "17.29",
		},
		{
			name: "no label",
			text: "some text without any amount label 12,34",
			want: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractAmount(tc.text); got != tc.want {
				t.Fatalf("extractAmount()=%q want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeAmount(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"1.234,56", "1234.56", true},
		{"1,234.56", "1234.56", true},
		{"1234.56", "1234.56", true},
		{"1234,56", "1234.56", true},
		{"1.234.567,89", "1234567.89", true},
		{"1,234,567.89", "1234567.89", true},
		{"1.234", "1234.00", true},
		{"12,5", "12.50", true},
		{"12", "12.00", true},
		{"", "0.00", false},
	}
	for _, tc := range tests {
		got, ok := normalizeAmount(tc.in)
		if ok != tc.ok {
			t.Fatalf("normalizeAmount(%q) ok=%v want %v", tc.in, ok, tc.ok)
		}
		if ok && got != tc.want {
			t.Fatalf("normalizeAmount(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
