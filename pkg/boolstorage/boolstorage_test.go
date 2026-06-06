package boolstorage

import (
	"testing"
)

func TestBoolStorageSet(t *testing.T) {
	storage := NewBoolStorage()
	if storage.Check() {
		t.Errorf("Expected initial value to be false")
	}
	storage.Set()
	if !storage.Check() {
		t.Errorf("Expected value to be true after Set()")
	}
}

func TestBoolStorageReset(t *testing.T) {
	storage := NewBoolStorage()
	storage.Set()
	storage.Reset()
	if storage.Check() {
		t.Errorf("Expected value to be false after Reset()")
	}
}

func TestBoolStorage256v256Set(t *testing.T) {
	storage := NewBoolStorage256v256()
	var id uint8 = 10
	var flow uint8 = 20
	if storage.Check(id, flow) {
		t.Errorf("Expected initial value to be false for id %d and flow %d", id, flow)
	}
	storage.Set(id, flow)
	if !storage.Check(id, flow) {
		t.Errorf("Expected value to be true after Set() for id %d and flow %d", id, flow)
	}
}

func TestBoolStorage256v256Reset(t *testing.T) {
	storage := NewBoolStorage256v256()
	var id uint8 = 10
	var flow uint8 = 20
	storage.Set(id, flow)
	storage.Reset(id, flow)
	if storage.Check(id, flow) {
		t.Errorf("Expected value to be false after Reset() for id %d and flow %d", id, flow)
	}
}
