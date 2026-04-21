//go:build !authembed

package main

func embeddedAuthSettings() authSettings {
	return authSettings{}
}
