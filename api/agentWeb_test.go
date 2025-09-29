package main

import "testing"

func TestAdd(t *testing.T) {
	str := "{\"a\":1}{\"b\":2}"
	ar, err := breakArgs(str)

	if err != nil {
		t.Errorf("error")
	}
	if len(ar) != 2 {
		t.Errorf("must be 2")
	}
}
