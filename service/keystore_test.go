// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016-2017 Canonical Ltd
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

package service

import "testing"

func TestGetKeyStoreFilesystem(t *testing.T) {
	// Set up the environment variables
	config := ConfigSettings{KeyStoreType: "filesystem", KeyStorePath: "../keystore"}
	Environ = &Env{Config: config}

	keystore, err := GetKeyStore(config)
	if err != nil {
		t.Error("Error setting up the filesystem keystore")
	}
	if keystore == nil {
		t.Error("Nil keystore returned")
	}
}

func TestGetKeyStoreInvalid(t *testing.T) {
	// Set up the environment variables
	config := ConfigSettings{KeyStoreType: "invalid", KeyStorePath: "../keystore"}
	Environ = &Env{Config: config}

	_, err := GetKeyStore(config)
	if err == nil {
		t.Errorf("Expected error, but got success: %v", err)
	}
}
