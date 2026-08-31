package export

import "testing"

// RFC 8785 section 3.2.2, using the published serialization sample.
func TestCanonicalizeJSONMatchesRFC8785Vector(t *testing.T) {
	t.Parallel()
	input := []byte(`{"numbers":[333333333.33333329,1E30,4.50,2e-3,0.000000000000000000000000001],"string":"€$\u000f\nA'B\"\\\"/","literals":[null,true,false]}`)
	want := `{"literals":[null,true,false],"numbers":[333333333.3333333,1e+30,4.5,0.002,1e-27],"string":"€$\u000f\nA'B\"\\\"/"}`
	got, err := canonicalizeJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("canonical JSON = %s, want %s", got, want)
	}
}
