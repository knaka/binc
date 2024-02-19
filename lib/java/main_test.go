package java

import "testing"

func Test_camel2Kebab(t *testing.T) {
	type args struct {
		sIn string
	}
	tests := []struct {
		name  string
		args  args
		wantS string
	}{
		{"0", args{"Camel"}, "camel"},
		{"1", args{"CamelCase"}, "camel-case"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotS := camel2Kebab(tt.args.sIn); gotS != tt.wantS {
				t.Errorf("camel2Kebab() = %v, want %v", gotS, tt.wantS)
			}
		})
	}
}

func Test_kebab2Camel(t *testing.T) {
	type args struct {
		sIn string
	}
	tests := []struct {
		name  string
		args  args
		wantS string
	}{
		{"0", args{"camel"}, "Camel"},
		{"1", args{"camel-case"}, "CamelCase"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotS := kebab2Camel(tt.args.sIn); gotS != tt.wantS {
				t.Errorf("kebab2Camel() = %v, want %v", gotS, tt.wantS)
			}
		})
	}
}
