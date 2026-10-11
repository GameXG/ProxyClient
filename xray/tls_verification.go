package xray

import (
	"fmt"
	"strings"
)

func validateTLSVerification(allowInsecure bool, pinnedPeerCertSha256, verifyPeerCertByName string) error {
	if !allowInsecure {
		return nil
	}
	if strings.TrimSpace(pinnedPeerCertSha256) != "" || strings.TrimSpace(verifyPeerCertByName) != "" {
		return nil
	}
	return fmt.Errorf("allowInsecure is no longer supported by bundled Xray; provide pcs/pinnedPeerCertSha256 or vcn/verifyPeerCertByName")
}

func applyTLSVerification(settings *TLSSettings, pinnedPeerCertSha256, verifyPeerCertByName string) {
	settings.PinnedPeerCertSha256 = strings.TrimSpace(pinnedPeerCertSha256)
	settings.VerifyPeerCertByName = strings.TrimSpace(verifyPeerCertByName)
}
