package errcode

import (
	"errors"
	"net/http"
	"testing"
)

func TestRegisterAndLookup(t *testing.T) {
	// 注册一个新的错误码(用高位避免与已注册的冲突)
	code := Register(9999, http.StatusTeapot, "测试错误")
	meta, ok := Lookup(code)
	if !ok {
		t.Fatal("Lookup should find registered code")
	}
	if meta.HTTPStatus != http.StatusTeapot {
		t.Errorf("HTTPStatus = %d, want %d", meta.HTTPStatus, http.StatusTeapot)
	}
	if meta.Message != "测试错误" {
		t.Errorf("Message = %q, want 测试错误", meta.Message)
	}
}

func TestLookupUnregistered(t *testing.T) {
	_, ok := Lookup(Code(88888))
	if ok {
		t.Error("Lookup should return false for unregistered code")
	}
}

func TestMustLookupFallback(t *testing.T) {
	meta := MustLookup(Code(88888))
	// 应降级到 CommonInternal
	if meta.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("expected fallback to 500, got %d", meta.HTTPStatus)
	}
}

func TestNewError(t *testing.T) {
	code := Register(9998, http.StatusBadRequest, "bad")
	cause := errors.New("underlying")
	err := New(code, cause)
	if err.Code != code {
		t.Errorf("Code = %d, want %d", err.Code, code)
	}
	if !errors.Is(err, cause) {
		t.Error("Unwrap should return cause")
	}
	if err.Error() == "" {
		t.Error("Error() should not be empty")
	}
}

func TestNewMsg(t *testing.T) {
	code := Register(9997, http.StatusBadRequest, "default")
	err := NewMsg(code, "custom message", nil)
	if err.Msg != "custom message" {
		t.Errorf("Msg = %q, want custom message", err.Msg)
	}
}

func TestRegisterPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("duplicate Register should panic")
		}
	}()
	code := Register(9996, http.StatusOK, "first")
	Register(code, http.StatusOK, "duplicate")
}
