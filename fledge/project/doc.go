// Package project provides utilities for detecting and working with
// Go projects and Firebird applications.
//
// # Overview
//
// This package helps CLI tools in the Firebird suite detect project
// information:
//   - Go module paths and versions (via go.mod)
//   - Firebird project detection (via firebird.yml)
//   - Project metadata and configuration
//
// # Usage
//
// Detect a Go module:
//
//	info, err := project.DetectModule(".")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Module: %s\n", info.Path)
//
// Check if a project is a Firebird app:
//
//	if project.IsFirebirdProject(".") {
//	    fmt.Println("This is a Firebird project!")
//	}
//
// Get Firebird configuration:
//
//	found, config, err := project.DetectFirebirdProject(".")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if found {
//	    fmt.Printf("Database: %s\n", config.Database)
//	    fmt.Printf("Router: %s\n", config.Router)
//	}
package project
