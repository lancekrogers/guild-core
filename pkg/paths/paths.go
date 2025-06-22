// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package paths provides centralized path constants and utilities for the Guild framework.
package paths

const (
	// DefaultCampaignDir is the default directory name for campaign-specific data
	DefaultCampaignDir = ".campaign"
	
	// DefaultGuildConfigFile is the default configuration file name
	DefaultGuildConfigFile = "guild.yaml"
	
	// DefaultMemoryDB is the default database file name
	DefaultMemoryDB = "memory.db"
	
	// CampaignHashFile is the binary hash file for ultra-fast detection
	CampaignHashFile = ".hash"
	
	// SocketRegistryFile contains campaign hash and metadata
	SocketRegistryFile = "socket-registry.yaml"
)