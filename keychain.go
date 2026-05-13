package main

import (
	"fmt"
	"strings"

	"github.com/keybase/go-keychain"
)

const (
	keychainService = "sonarsweep"
	keychainAccount = "user_token"
	keychainLabel   = "SonarSweep User Token"
)

func StoreToken(token string) error {
	item := keychain.NewItem()
	item.SetService(keychainService)
	item.SetAccount(keychainAccount)
	item.SetLabel(keychainLabel)
	item.SetData([]byte(token))
	item.SetAccessible(keychain.AccessibleWhenUnlocked)

	keychain.DeleteItem(item)

	err := keychain.AddItem(item)
	if err != nil {
		return fmt.Errorf("failed to store token in keychain: %w", err)
	}
	return nil
}

func GetToken() (string, error) {
	data, err := keychain.GetGenericPassword(keychainService, keychainAccount, keychainLabel, "")
	if err != nil {
		if strings.Contains(err.Error(), "Item not found") || strings.Contains(err.Error(), "secItemNotFound") {
			return "", nil
		}
		return "", fmt.Errorf("failed to get token from keychain: %w", err)
	}
	return string(data), nil
}

func DeleteToken() error {
	err := keychain.DeleteGenericPasswordItem(keychainService, keychainAccount)
	if err != nil {
		if strings.Contains(err.Error(), "Item not found") || strings.Contains(err.Error(), "secItemNotFound") {
			return nil
		}
		return fmt.Errorf("failed to delete token from keychain: %w", err)
	}
	return nil
}