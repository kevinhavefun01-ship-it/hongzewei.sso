package domain

import "testing"

func TestSetPasswordAndCheck(t *testing.T) {
	u := &User{Username: "test"}
	if err := u.SetPassword("hello123"); err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}
	if !u.CheckPassword("hello123") {
		t.Fatal("CheckPassword should return true for correct password")
	}
	if u.CheckPassword("wrong") {
		t.Fatal("CheckPassword should return false for wrong password")
	}
}

func TestCanLogin(t *testing.T) {
	tests := []struct {
		name     string
		user     *User
		wantOK   bool
		wantMsg  bool
	}{
		{"正常用户", &User{IsActive: true, IsLocked: false}, true, false},
		{"已禁用", &User{IsActive: false, IsLocked: false}, false, true},
		{"已锁定", &User{IsActive: true, IsLocked: true}, false, true},
		{"禁用+锁定", &User{IsActive: false, IsLocked: true}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, msg := tt.user.CanLogin()
			if ok != tt.wantOK {
				t.Errorf("CanLogin() ok = %v, want %v", ok, tt.wantOK)
			}
			if tt.wantMsg && msg == "" {
				t.Error("expected non-empty error message")
			}
		})
	}
}

func TestLockUnlock(t *testing.T) {
	u := &User{}
	u.Lock()
	if !u.IsLocked || u.LockedAt == nil {
		t.Fatal("Lock() should set IsLocked=true and LockedAt")
	}
	ok, _ := u.CanLogin()
	if ok {
		t.Fatal("locked user should not be able to login")
	}
	u.Unlock()
	if u.IsLocked || u.LockedAt != nil {
		t.Fatal("Unlock() should reset IsLocked and LockedAt")
	}
}

func TestRecordLogin(t *testing.T) {
	u := &User{}
	u.RecordLogin("1.2.3.4", "password")
	if u.LastLoginIP != "1.2.3.4" {
		t.Errorf("LastLoginIP = %q, want 1.2.3.4", u.LastLoginIP)
	}
	if u.LastLoginMethod != "password" {
		t.Errorf("LastLoginMethod = %q, want password", u.LastLoginMethod)
	}
	if u.LastLoginAt == nil {
		t.Fatal("LastLoginAt should not be nil")
	}
}

func TestMarkPasswordChanged(t *testing.T) {
	u := &User{MustChangePassword: true}
	u.MarkPasswordChanged()
	if u.MustChangePassword {
		t.Fatal("MarkPasswordChanged should clear MustChangePassword")
	}
}

func TestTableName(t *testing.T) {
	u := &User{}
	if u.TableName() != "sso_users" {
		t.Errorf("TableName() = %q, want sso_users", u.TableName())
	}
}
