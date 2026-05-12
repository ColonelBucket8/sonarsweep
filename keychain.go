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
	existing, _ := GetToken()
	if existing != "" {
		item := keychain.NewGenericPassword(keychainService, keychainAccount, keychainLabel, []byte(token), "")
		err := keychain.UpdateItem(item, item)
		if err != nil {
			return fmt.Errorf("failed to update token in keychain: %w", err)
		}
		return nil
	}

	item := keychain.NewGenericPassword(keychainService, keychainAccount, keychainLabel, []byte(token), "")
	err := keychain.AddItem(item)
	if err != nil {
		return fmt.Errorf("failed to store token in keychain: %w", err)
	}
	return nil
}

func GetToken() (string, error) {
	data, err := keychain.GetGenericPassword(keychainService, keychainAccount, "", "")
	if err != nil {
		if strings.Contains(err.Error(), "Item not found") {
			return "", nil
		}
		return "", fmt.Errorf("failed to get token from keychain: %w", err)
	}
	return string(data), nil
}

func DeleteToken() error {
	err := keychain.DeleteGenericPasswordItem(keychainService, keychainAccount)
	if err != nil {
		if strings.Contains(err.Error(), "Item not found") {
			return nil
		}
		return fmt.Errorf("failed to delete token from keychain: %w", err)
	}
	return nil
}

