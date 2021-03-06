// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2018 Canonical Ltd
 * License granted by Canonical Limited
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package sync

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/CanonicalLtd/serial-vault/crypt"
	"github.com/CanonicalLtd/serial-vault/datastore"
	"github.com/CanonicalLtd/serial-vault/service/keypair"
	"github.com/CanonicalLtd/serial-vault/service/log"
)

// Client is the sync interface for the serial vault
type Client interface {
	Accounts() error
}

// FactoryClient is the implementation of the factory sync for the serial vault
type FactoryClient struct {
	URL      string
	Username string
	APIKey   string
}

// NewFactoryClient creates a factory client to sync data with the cloud serial-vault
func NewFactoryClient(url, username, apiKey string) *FactoryClient {
	hclient = http.Client{}
	return &FactoryClient{
		URL: url, Username: username, APIKey: apiKey,
	}
}

// Accounts synchronizes the account details to the factory instance
func (c *FactoryClient) Accounts() error {
	// Fetch the accounts from the serial-vault
	result, err := FetchAccounts(c.URL, c.Username, c.APIKey)
	if err != nil {
		log.Errorf("Error parsing accounts: %v", err)
		return err
	}
	if !result.Success {
		log.Errorf("Error fetching accounts: %s", result.ErrorMessage)
		return errors.New(result.ErrorMessage)
	}

	// Update the factory database with the accounts
	for _, a := range result.Accounts {
		if err = datastore.Environ.DB.SyncAccount(a); err != nil {
			log.Errorf("Error updating accounts: %v", err)
			return err
		}
	}

	return nil
}

// SigningKeys synchronizes the signing-keys to the factory instance
func (c *FactoryClient) SigningKeys() error {
	// Get the signing keys by sending our keystore secret
	req := keypair.SyncRequest{Secret: datastore.Environ.Config.KeyStoreSecret}
	data, err := json.Marshal(req)
	if err != nil {
		log.Errorf("Error with keystore secret: %v", err)
		return err
	}

	// Fetch the signing-keys from the cloud serial-vault
	result, err := FetchSigningKeys(c.URL, c.Username, c.APIKey, data)
	if err != nil {
		log.Errorf("Error parsing signing-keys: %v", err)
		return err
	}
	if !result.Success {
		log.Errorf("Error fetching signing-keys")
		return errors.New("Error fetching signing keys")
	}

	// Update the factory database with the signing-keys
	for _, k := range result.Keypairs {

		// Check if we've already sync-ed the keypair
		_, err = GetKeypairByPublicID(k.AuthorityID, k.KeyID)
		if err == nil {
			// Already have the keypair, so no need to store it again
			// This is important as we get a new encryption key and sealed key each time
			continue
		}

		err = datastore.Environ.DB.SyncKeypair(k)
		if err != nil {
			log.Errorf("Error updating keypairs: %v", err)
			return err
		}

		err = datastore.Environ.DB.PutSetting(
			datastore.Setting{
				Code: crypt.GenerateAuthKey(k.AuthorityID, k.KeyID),
				Data: k.AuthKeyHash})
		if err != nil {
			log.Errorf("Error saving keypair auth: %v", err)
			return err
		}
	}

	return nil
}

// Models synchronizes the model details to the factory instance
func (c *FactoryClient) Models() error {
	// Fetch the accounts from the serial-vault
	result, err := FetchModels(c.URL, c.Username, c.APIKey)
	if err != nil {
		log.Errorf("Error parsing models: %v", err)
		return err
	}
	if !result.Success {
		log.Errorf("Error fetching models: %s", result.ErrorMessage)
		return errors.New(result.ErrorMessage)
	}

	// Update the factory database with the accounts
	for _, m := range result.Models {
		err = datastore.Environ.DB.SyncModel(m)
		if err != nil {
			log.Errorf("Error updating models: %v", err)
			return err
		}

	}

	return nil
}

// SigningLogs sends signing logs to the cloud from the factory
func (c *FactoryClient) SigningLogs() error {
	// Fetch the signing logs that have not been synced
	logs, err := datastore.Environ.DB.SyncSigningLog()
	if err != nil {
		log.Errorf("Error fetching unsynced signing logs: %v", err)
		return err
	}

	// Send each signing log to the cloud
	for _, l := range logs {
		success, err := SendSigningLog(c.URL, c.Username, c.APIKey, l)
		if err != nil || !success {
			// Leave this one till the next sync
			continue
		}

		// Mark the sync as done
		err = datastore.Environ.DB.SyncUpdateSigningLog(l.ID)
		if err != nil {
			log.Errorf("Error marking signing logs: %v", err)
		}
	}

	return nil
}

// TestLogs sends logs to the cloud from the factory
func (c *FactoryClient) TestLogs() error {
	// Fetch the test logs that have not been synced
	logs, err := datastore.Environ.DB.SyncListTestLogs()
	if err != nil {
		log.Errorf("Error fetching unsynced test logs: %v", err)
		return err
	}

	// Send each signing log to the cloud
	for _, l := range logs {
		success, err := SendTestLog(c.URL, c.Username, c.APIKey, l)
		if err != nil || !success {
			// Leave this one till the next sync
			continue
		}

		// Delete the factory test log
		err = datastore.Environ.DB.SyncDeleteTestLog(l.ID)
		if err != nil {
			log.Errorf("Error deleting test log: %v", err)
		}
	}

	return nil
}

// GetKeypairByPublicID is the mockable call to the database function
var GetKeypairByPublicID = func(authorityID, keyID string) (datastore.Keypair, error) {
	return datastore.Environ.DB.GetKeypairByPublicID(authorityID, keyID)
}
