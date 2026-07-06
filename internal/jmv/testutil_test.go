package jmv

import "testing"

func disableProfileMutation(t *testing.T) {
	t.Helper()
	t.Setenv("JMV_NO_MODIFY_PROFILE", "1")
}
